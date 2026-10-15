package plant

import (
	"math"
	"testing"
)

func TestAnalisarDegrau(t *testing.T) {
	// λ sobe para 1,25, volta, passa até 0,99 e assenta em 1,00.
	var ts, ys []float64
	for i := 0; i <= 400; i++ {
		tt := float64(i) * 0.01
		y := 1.0
		switch {
		case tt >= 1 && tt < 1.5:
			y = 1.25
		case tt >= 1.5 && tt < 2:
			y = 1.10
		case tt >= 2 && tt < 2.5:
			y = 0.99
		case tt >= 2.5:
			y = 1.001
		}
		ts, ys = append(ts, tt), append(ys, y)
	}
	d := AnalisarDegrau(ts, ys, 1.0, 1.0, 0.02)
	if math.Abs(d.DesvioMax-0.25) > 1e-9 {
		t.Errorf("DesvioMax = %v", d.DesvioMax)
	}
	if math.Abs(d.Sobressinal-0.04) > 1e-9 {
		t.Errorf("Sobressinal = %v, quer 0,04", d.Sobressinal)
	}
	if math.Abs(d.Acomodacao-0.99) > 1e-9 { // última amostra fora: t = 1,99
		t.Errorf("Acomodacao = %v", d.Acomodacao)
	}
	if !d.Acomodou || math.Abs(d.Final-1.001) > 1e-9 {
		t.Errorf("Final = %v, Acomodou = %v", d.Final, d.Acomodou)
	}
}

func TestAnalisarDegrauSemDesvio(t *testing.T) {
	d := AnalisarDegrau([]float64{0, 1, 2}, []float64{1, 1, 1}, 0.5, 1, 0.02)
	if d.DesvioMax != 0 || d.Sobressinal != 0 || d.Acomodacao != 0 || !d.Acomodou {
		t.Errorf("%+v", d)
	}
}

func TestAnalisarDegrauIgnoraRuidoAntesDoPico(t *testing.T) {
	// desvio inicial minúsculo no sentido oposto não define o sentido do degrau
	ts := []float64{0, 1, 2, 3, 4}
	ys := []float64{1, 0.9999, 1.2, 1.0, 1.0}
	d := AnalisarDegrau(ts, ys, 0, 1, 0.02)
	if d.Sobressinal != 0 {
		t.Errorf("Sobressinal = %v, quer 0", d.Sobressinal)
	}
}
