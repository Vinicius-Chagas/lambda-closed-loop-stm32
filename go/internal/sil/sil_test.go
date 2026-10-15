package sil

import (
	"testing"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/engine"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/plant"
)

// TPS que leva o motor a 2000 rpm em regime (engine.RPMAlvo).
var tps2000 = (2000 - engine.RPMLenta) / (engine.RPMMax - engine.RPMLenta) * 100

const (
	tDegrau = 6.0 // s: tempo para a rotação estabilizar em 2000 rpm
	tFim    = 9.0
	banda   = 0.02
)

type ensaio struct {
	t, sonda []float64
	ecu      *ECU
	ultima   plant.Sample
}

// degrau roda a bancada em 2000 rpm e aplica a perturbação em tDegrau.
func degrau(t *testing.T, modo int, perturbacao plant.Command) ensaio {
	return degrauEm(t, tps2000, modo, perturbacao)
}

func degrauEm(t *testing.T, tps float64, modo int, perturbacao plant.Command) ensaio {
	t.Helper()
	ecu := New()
	ecu.SetModo(modo)
	p := plant.New(plant.Config{}, ecu)
	defer p.Close()
	p.Apply(plant.Command{TPS: &tps})

	var r ensaio
	r.ecu = ecu
	aplicado := false
	for {
		s, err := p.Step()
		if err != nil {
			t.Fatal(err)
		}
		if !aplicado && s.Tempo >= tDegrau-1e-9 {
			p.Apply(perturbacao)
			aplicado = true
		}
		r.t = append(r.t, s.Tempo)
		r.sonda = append(r.sonda, s.LambdaSonda)
		r.ultima = s
		if s.Tempo >= tFim {
			return r
		}
	}
}

func ptr(v float64) *float64 { return &v }

func TestSILDegrauMalhaFechada(t *testing.T) {
	casos := []struct {
		nome string
		cmd  plant.Command
	}{
		{"bico −20 %", plant.Command{FatorBico: ptr(0.8)}},
		{"bico +20 %", plant.Command{FatorBico: ptr(1.2)}},
		{"vazamento 10 mg", plant.Command{VazamentoMg: ptr(10)}},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			r := degrau(t, MalhaFechada, c.cmd)
			if rpm := r.ultima.RPM; rpm < 1990 || rpm > 2010 {
				t.Fatalf("rotação %.0f, quer 2000", rpm)
			}
			// Antes do degrau já está em λ = 1 (nada fora da banda no último 1 s).
			n := 0
			for n < len(r.t) && r.t[n] < tDegrau {
				n++
			}
			antes := plant.AnalisarDegrau(r.t[:n], r.sonda[:n], tDegrau-1, 1, banda)
			if antes.DesvioMax > banda {
				t.Errorf("λ fora da banda antes do degrau (%.3f)", antes.DesvioMax)
			}
			d := plant.AnalisarDegrau(r.t, r.sonda, tDegrau, 1, banda)
			t.Logf("desvio máx %.3f, sobressinal %.1f %%, acomodação %.2f s, λ final %.4f, correção %+.1f %%",
				d.DesvioMax, d.Sobressinal*100, d.Acomodacao, d.Final, r.ecu.Saida().Correcao*100)
			if d.DesvioMax < 0.05 {
				t.Errorf("o degrau quase não apareceu (%.3f)", d.DesvioMax)
			}
			if !d.Acomodou || d.Acomodacao > 2.0 {
				t.Errorf("acomodação %.2f s (meta ≤ 2 s)", d.Acomodacao)
			}
			if d.Sobressinal > 0.05 {
				t.Errorf("sobressinal %.1f %% (meta ≤ 5 %%)", d.Sobressinal*100)
			}
			if c := r.ecu.Saida().Correcao; c >= 0.299 || c <= -0.299 {
				t.Errorf("correção %.3f encostada no limite", c)
			}
		})
	}
}

// Pior caso para a estabilidade: marcha lenta, com o maior atraso de
// transporte (245 ms). A meta de 2 s é em 2000 rpm; aqui só se exige que
// estabilize sem oscilar.
func TestSILMarchaLentaEstavel(t *testing.T) {
	r := degrauEm(t, 0, MalhaFechada, plant.Command{FatorBico: ptr(0.8)})
	d := plant.AnalisarDegrau(r.t, r.sonda, tDegrau, 1, banda)
	t.Logf("lenta (%.0f rpm): desvio máx %.3f, sobressinal %.1f %%, acomodação %.2f s, λ final %.4f",
		r.ultima.RPM, d.DesvioMax, d.Sobressinal*100, d.Acomodacao, d.Final)
	if !d.Acomodou || d.Acomodacao > 2.5 || d.Sobressinal > 0.10 {
		t.Errorf("resposta em marcha lenta: %+v", d)
	}
}

func TestSILMalhaAbertaNaoCorrige(t *testing.T) {
	r := degrau(t, MalhaAberta, plant.Command{FatorBico: ptr(0.8)})
	d := plant.AnalisarDegrau(r.t, r.sonda, tDegrau, 1, banda)
	// Bico com 80 % da vazão: λ ≈ 1/0,8 = 1,25 e fica lá.
	if d.Final < 1.2 || d.Final > 1.3 {
		t.Errorf("λ final %.3f, quer ≈ 1,25", d.Final)
	}
	if c := r.ecu.Saida().Correcao; c != 0 {
		t.Errorf("correção %v em malha aberta", c)
	}
}

func TestSILComandosDoPainel(t *testing.T) {
	ecu := New()
	ecu.SetAlvo(1.5) // fora da faixa: ignorado
	ecu.SetModo(7)   // inválido: ignorado
	p := plant.New(plant.Config{}, ecu)
	defer p.Close()
	ecu.SetAlvo(1.1) // λ alvo pobre
	var s plant.Sample
	var err error
	for range 1000 { // 5 s
		if s, err = p.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if s.LambdaSonda < 1.08 || s.LambdaSonda > 1.12 {
		t.Errorf("λ sonda %.3f, quer 1,10", s.LambdaSonda)
	}
	if ecu.Passos() != 500 {
		t.Errorf("%d passos de controle em 5 s, quer 500", ecu.Passos())
	}
}

func TestSILRejeitaEntradaInvalida(t *testing.T) {
	ecu := New()
	if _, err := ecu.Step([]float64{1, 2}, 0.005); err == nil {
		t.Error("aceitou 2 tensões")
	}
	if _, err := ecu.Step([]float64{1, 2, 3}, 0.02); err == nil {
		t.Error("aceitou dt maior que o período de controle")
	}
}
