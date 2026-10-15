package painel

import (
	"math"
	"testing"
	"time"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/canbus"
)

var t0 = time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

func planta(lambda float64, bico, vazamento uint8) canbus.PlantaEstado {
	return canbus.PlantaEstado{LambdaCilindro: lambda, LambdaSonda: lambda, VazaoBicoPct: bico, VazamentoMg: vazamento}
}

func TestHistoricoGuardaSoAJanela(t *testing.T) {
	var e Estado
	e.Atualizar(canbus.EcuEstado{LambdaAlvo: 1}.Frame(), t0)
	for i := range 1500 { // 30 s simulados
		e.Atualizar(planta(1+float64(i)/10000, 100, 0).Frame(), t0)
	}
	h := e.Historico()
	if n := len(h); n != 501 { // 10 s / 20 ms + o ponto inicial
		t.Fatalf("%d amostras, quer 501", n)
	}
	if fim := h[len(h)-1].T; math.Abs(fim-30) > 1e-9 || math.Abs(fim-h[0].T-Janela) > 1e-9 {
		t.Errorf("janela de %.3f a %.3f", h[0].T, fim)
	}
	if h[0].LambdaAlvo != 1 {
		t.Errorf("alvo %v", h[0].LambdaAlvo)
	}
	if len(e.hist) > 1001 {
		t.Errorf("histórico cresce sem limite: %d", len(e.hist))
	}
}

func TestConectado(t *testing.T) {
	var e Estado
	if e.Conectado(t0) {
		t.Error("conectado sem frames")
	}
	e.Atualizar(canbus.EcuSensores{RPM: 800}.Frame(), t0)
	if !e.Conectado(t0.Add(time.Second)) || e.Conectado(t0.Add(3*time.Second)) {
		t.Error("estado de conexão errado")
	}
}

func TestComandos(t *testing.T) {
	var e Estado
	e.Atualizar(canbus.EcuSensores{RPM: 2000, TPS: 23.1}.Frame(), t0)
	e.Atualizar(canbus.EcuEstado{Modo: canbus.MalhaFechada, LambdaAlvo: 1}.Frame(), t0)
	e.Atualizar(planta(1, 100, 0).Frame(), t0)

	cmd := func(k Tecla) canbus.PainelPlanta {
		m, err := canbus.DecodePainelPlanta(e.Comando(k))
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	if m := cmd(AceleradorMais); m.Acelerador != 30 || m.AjusteBico != canbus.ManterBico || m.VazamentoMg != canbus.ManterVazamento {
		t.Errorf("acelerador + : %+v", m)
	}
	if m := cmd(AceleradorMenos); m.Acelerador != 20 {
		t.Errorf("acelerador − : %+v", m)
	}
	if m := cmd(AlternarBico); m.AjusteBico != PerturbacaoBico || m.Acelerador != canbus.ManterAcelerador {
		t.Errorf("bico: %+v", m)
	}
	if m := cmd(AlternarVazamento); m.VazamentoMg != PerturbacaoVazamento {
		t.Errorf("vazamento: %+v", m)
	}
	f := e.Comando(AlternarMalha)
	if f.ID != canbus.IDPainelEcu || f.Data[2] != canbus.MalhaAberta || f.Data[0] != 0 || f.Data[1] != 0 {
		t.Errorf("malha: %v", f)
	}

	// Com as perturbações ligadas, as teclas desligam.
	e.Atualizar(planta(1.2, 80, 10).Frame(), t0)
	e.Atualizar(canbus.EcuEstado{Modo: canbus.MalhaAberta}.Frame(), t0)
	if m := cmd(AlternarBico); m.AjusteBico != 0 {
		t.Errorf("bico desligar: %+v", m)
	}
	if m := cmd(AlternarVazamento); m.VazamentoMg != 0 {
		t.Errorf("vazamento desligar: %+v", m)
	}
	if f := e.Comando(AlternarMalha); f.Data[2] != canbus.MalhaFechada {
		t.Errorf("malha: %v", f)
	}

	// Acelerador nos extremos.
	e.Atualizar(canbus.EcuSensores{TPS: 100}.Frame(), t0)
	if m := cmd(AceleradorMais); m.Acelerador != 100 {
		t.Errorf("acima de 100: %+v", m)
	}
	e.Atualizar(canbus.EcuSensores{TPS: 0}.Frame(), t0)
	if m := cmd(AceleradorMenos); m.Acelerador != 0 {
		t.Errorf("abaixo de 0: %+v", m)
	}
}

func TestClassificarEBarra(t *testing.T) {
	casos := []struct {
		lambda float64
		m      Mistura
		pos    float64
	}{
		{0.70, Rica, 0}, {0.90, Rica, 0.25}, {0.985, Ideal, 0.4625}, {1.0, Ideal, 0.5},
		{1.019, Ideal, 0.5475}, {1.10, Pobre, 0.75}, {1.5, Pobre, 1},
	}
	for _, c := range casos {
		if m := Classificar(c.lambda, 1); m != c.m {
			t.Errorf("λ %.3f: %v, quer %v", c.lambda, m, c.m)
		}
		if p := PosicaoBarra(c.lambda); math.Abs(p-c.pos) > 1e-9 {
			t.Errorf("λ %.3f: posição %v, quer %v", c.lambda, p, c.pos)
		}
	}
	if Classificar(1.05, 1.05) != Ideal || Rica.String() != "RICA" {
		t.Error("alvo diferente de 1")
	}
}
