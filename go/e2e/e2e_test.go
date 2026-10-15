//go:build e2e

// Teste ponta a ponta: ECU (firmware no Renode) + planta (Go) conversando
// pelo vcan0, como na feira. Só observa e comanda pelo CAN, como o painel.
// Precisa da bancada no ar: scripts/e2e.sh (ou make test-e2e) cuida disso.
package e2e

import (
	"context"
	"flag"
	"math"
	"testing"
	"time"

	"go.einride.tech/can"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/canbus"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/plant"
)

var iface = flag.String("can", "vcan0", "interface SocketCAN da bancada")

type bancada struct {
	t      *testing.T
	bus    *canbus.Bus
	frames chan can.Frame
	v      canbus.Visao
}

func abrir(t *testing.T) *bancada {
	bus, err := canbus.Open(context.Background(), *iface)
	if err != nil {
		t.Fatalf("abrindo %s: %v", *iface, err)
	}
	b := &bancada{t: t, bus: bus, frames: make(chan can.Frame, 4096)}
	go func() {
		for {
			f, ok := bus.Receive()
			if !ok {
				close(b.frames)
				return
			}
			b.frames <- f
		}
	}()
	t.Cleanup(func() { bus.Close() })
	return b
}

func (b *bancada) enviar(f can.Frame) {
	b.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.bus.Send(ctx, f); err != nil {
		b.t.Fatalf("enviando 0x%X: %v", f.ID, err)
	}
}

// ate consome frames até cond ser verdadeira (cond é avaliada a cada frame
// conhecido, depois de atualizar a visão). Falha após timeout de relógio.
func (b *bancada) ate(oQue string, timeout time.Duration, cond func(f can.Frame) bool) {
	b.t.Helper()
	limite := time.After(timeout)
	for {
		select {
		case f, ok := <-b.frames:
			if !ok {
				b.t.Fatalf("barramento fechou esperando %s", oQue)
			}
			if b.v.Atualizar(f) && cond(f) {
				return
			}
		case <-limite:
			b.t.Fatalf("timeout de %v esperando %s; última visão: %+v", timeout, oQue, b.v)
		}
	}
}

// simulado consome frames por d segundos de tempo simulado (contados pelos
// frames 0x300) e chama cada para cada frame 0x300.
func (b *bancada) simulado(d float64, cada func()) {
	b.t.Helper()
	fim := b.v.NPlanta + int(math.Round(d/canbus.PeriodoPlanta))
	b.ate("tempo simulado", time.Duration(d*10+30)*time.Second, func(f can.Frame) bool {
		if f.ID == canbus.IDPlanta && cada != nil {
			cada()
		}
		return b.v.NPlanta >= fim
	})
}

func (b *bancada) painelPlanta(acel uint8, bico int8, vazamento uint8) {
	b.enviar(canbus.PainelPlanta{Acelerador: acel, AjusteBico: bico, VazamentoMg: vazamento}.Frame())
}

func (b *bancada) painelEcu(modo uint8, zerarTempos bool) {
	b.enviar(canbus.PainelEcu{Modo: modo, ZerarTempos: zerarTempos}.Frame())
}

func TestBancada(t *testing.T) {
	b := abrir(t)

	t.Run("ECU e planta publicam no barramento", func(t *testing.T) {
		b.t = t
		b.ate("0x100, 0x101, 0x110 e 0x300", 120*time.Second, func(can.Frame) bool {
			return b.v.NEstado > 5 && b.v.NSensores > 5 && b.v.NTempos > 0 && b.v.NPlanta > 5
		})
		if b.v.Estado.Modo != canbus.MalhaFechada || b.v.Estado.LambdaAlvo != 1 {
			t.Errorf("estado inicial %+v", b.v.Estado)
		}
	})

	t.Run("acelera para 2000 rpm com λ = 1", func(t *testing.T) {
		b.t = t
		b.painelPlanta(23, canbus.ManterBico, 0) // 800 + 5200·0,23 = 1996 rpm
		estavel := 0
		b.ate("2000 rpm e λ estável", 120*time.Second, func(f can.Frame) bool {
			if f.ID != canbus.IDPlanta {
				return false
			}
			if math.Abs(b.v.Sensores.RPM-1996) < 15 && math.Abs(b.v.Planta.LambdaSonda-1) <= 0.01 {
				estavel++
			} else {
				estavel = 0
			}
			return estavel >= 50 // 1 s simulado
		})
		// A ECU mede o que a sonda da planta manda (ADC + conversão).
		if d := math.Abs(b.v.Estado.LambdaMedido - b.v.Planta.LambdaSonda); d > 0.01 {
			t.Errorf("λ ECU %.3f × sonda %.3f", b.v.Estado.LambdaMedido, b.v.Planta.LambdaSonda)
		}
	})

	t.Run("bico −20 %: malha fechada corrige", func(t *testing.T) {
		b.t = t
		b.painelEcu(canbus.ManterModo, true) // mede os tempos só a partir daqui
		b.painelPlanta(canbus.ManterAcelerador, -20, canbus.ManterVazamento)
		var ts, ys []float64
		t0 := math.NaN()
		b.simulado(4, func() {
			if math.IsNaN(t0) && b.v.Planta.VazaoBicoPct == 80 {
				t0 = b.v.TempoPlanta()
			}
			ts, ys = append(ts, b.v.TempoPlanta()), append(ys, b.v.Planta.LambdaSonda)
		})
		if math.IsNaN(t0) {
			t.Fatalf("a planta não aplicou o degrau: %+v", b.v.Planta)
		}
		d := plant.AnalisarDegrau(ts, ys, t0, 1, 0.02)
		t.Logf("desvio máx %.3f, sobressinal %.1f %%, acomodação %.2f s, λ final %.4f, correção %+.1f %%",
			d.DesvioMax, d.Sobressinal*100, d.Acomodacao, d.Final, b.v.Estado.Correcao*100)
		if d.DesvioMax < 0.1 {
			t.Errorf("o degrau quase não apareceu (%.3f)", d.DesvioMax)
		}
		if !d.Acomodou || d.Acomodacao > 2 {
			t.Errorf("acomodação %.2f s (meta ≤ 2 s)", d.Acomodacao)
		}
		if d.Sobressinal > 0.05 {
			t.Errorf("sobressinal %.1f %% (meta ≤ 5 %%)", d.Sobressinal*100)
		}
	})

	t.Run("ControlTask cumpre os tempos", func(t *testing.T) {
		b.t = t
		n := b.v.NTempos
		b.ate("0x110", 30*time.Second, func(can.Frame) bool { return b.v.NTempos > n })
		tm := b.v.Tempos
		t.Logf("período %d µs, jitter máx %d µs, execução máx %d µs, deadlines perdidos %d",
			tm.PeriodoUs, tm.JitterMaxUs, tm.ExecMaxUs, tm.DeadlinesPerdidos)
		if math.Abs(float64(tm.PeriodoUs)-10000) > 100 || tm.JitterMaxUs > 100 ||
			tm.ExecMaxUs == 0 || tm.ExecMaxUs > 1000 || tm.DeadlinesPerdidos != 0 {
			t.Errorf("tempos fora da meta: %+v", tm)
		}
	})

	t.Run("malha aberta não corrige", func(t *testing.T) {
		b.t = t
		b.painelEcu(canbus.MalhaAberta, false)
		b.ate("modo aberto", 30*time.Second, func(f can.Frame) bool {
			return f.ID == canbus.IDEcuEstado && b.v.Estado.Modo == canbus.MalhaAberta
		})
		b.simulado(2, nil)
		if l := b.v.Planta.LambdaSonda; l < 1.2 || l > 1.3 {
			t.Errorf("λ %.3f em malha aberta com o bico −20 %%, quer ≈ 1,25", l)
		}
		if b.v.Estado.Correcao != 0 {
			t.Errorf("correção %.3f em malha aberta", b.v.Estado.Correcao)
		}
	})

	t.Run("volta à malha fechada", func(t *testing.T) {
		b.t = t
		b.painelEcu(canbus.MalhaFechada, false)
		b.simulado(3, nil)
		if l := b.v.Planta.LambdaSonda; math.Abs(l-1) > 0.02 {
			t.Errorf("λ %.3f depois de voltar à malha fechada", l)
		}
	})
}
