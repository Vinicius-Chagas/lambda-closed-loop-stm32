package plant

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/engine"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/renode"
)

// ecuFixa responde sempre com o mesmo pulso e guarda o que recebeu.
type ecuFixa struct {
	pulso  float64
	volts  []float64
	tempo  float64
	passos int
	falha  error
}

func (e *ecuFixa) Step(v []float64, dt float64) (renode.Injector, error) {
	e.volts = v
	e.tempo += dt
	e.passos++
	return renode.Injector{PulseUs: e.pulso, PeriodUs: 150000}, e.falha
}

func ptr(x float64) *float64 { return &x }

func TestMalhaAbertaEstequiometrica(t *testing.T) {
	ti := engine.TiEstequiometrico(engine.MassaAr(0, engine.RPMLenta))
	ecu := &ecuFixa{pulso: ti}
	p := New(Config{}, ecu)
	defer p.Close()

	var s Sample
	var err error
	for range 400 { // 2 s
		if s, err = p.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if math.Abs(ecu.tempo-2.0) > 1e-9 || math.Abs(s.Tempo-2.0) > 1e-9 {
		t.Fatalf("tempos fora de sincronia: ECU %.4f s, planta %.4f s", ecu.tempo, s.Tempo)
	}
	if math.Abs(s.LambdaSonda-1) > 0.002 {
		t.Errorf("λ sonda = %.4f, esperado 1", s.LambdaSonda)
	}
	if math.Abs(ecu.volts[0]-1.65) > 0.01 {
		t.Errorf("tensão da sonda = %.3f V, esperado 1,65", ecu.volts[0])
	}
}

func TestComandosDoPainel(t *testing.T) {
	ti := engine.TiEstequiometrico(engine.MassaAr(0, engine.RPMLenta))
	p := New(Config{}, &ecuFixa{pulso: ti})
	defer p.Close()

	p.Apply(Command{FatorBico: ptr(0.8)})
	var s Sample
	for range 400 {
		s, _ = p.Step()
	}
	// Sem correção da ECU, bico com 80 % da vazão → λ = 1,25.
	if math.Abs(s.LambdaSonda-1.25) > 0.005 || s.FatorBico != 0.8 {
		t.Errorf("λ sonda = %.4f (fator %.2f), esperado 1,25", s.LambdaSonda, s.FatorBico)
	}

	p.Apply(Command{TPS: ptr(50)})
	for range 600 {
		s, _ = p.Step()
	}
	if s.TPS != 50 || math.Abs(s.RPM-engine.RPMAlvo(50)) > 50 {
		t.Errorf("TPS %.0f, RPM %.0f", s.TPS, s.RPM)
	}
}

func TestErroDaECU(t *testing.T) {
	p := New(Config{}, &ecuFixa{falha: errors.New("caiu")})
	defer p.Close()
	if _, err := p.Step(); err == nil {
		t.Fatal("esperado erro")
	}
}

func TestCSV(t *testing.T) {
	p := New(Config{}, &ecuFixa{pulso: 1600})
	defer p.Close()
	var buf bytes.Buffer
	w, err := NewCSVWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		s, _ := p.Step()
		w.Write(s)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 || !strings.HasPrefix(lines[0], "tempo_s,") || !strings.HasPrefix(lines[3], "0.015,") {
		t.Errorf("CSV:\n%s", buf.String())
	}
}
