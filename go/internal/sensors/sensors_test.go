package sensors

import (
	"math"
	"testing"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/engine"
)

func near(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %.5f, esperado %.5f ± %g", name, got, want, tol)
	}
}

func TestEscalas(t *testing.T) {
	near(t, "λ 0,7", LambdaVolts(0.7), 0, 1e-9)
	near(t, "λ 1,0", LambdaVolts(1.0), 1.65, 1e-9)
	near(t, "λ 1,3", LambdaVolts(1.3), 3.3, 1e-9)
	near(t, "λ 2,0 satura", LambdaVolts(2.0), 3.3, 0)
	near(t, "TPS 0", TPSVolts(0), 0.3, 1e-9)
	near(t, "TPS 100", TPSVolts(100), 3.0, 1e-9)
	near(t, "RPM 4000", RPMVolts(4000), 1.65, 1e-9)
	if c := Counts(1.65); c != 2048 {
		t.Errorf("Counts(1,65) = %d, esperado 2048 (mesmo valor dos testes Robot)", c)
	}
}

func TestSondaPrimeiraOrdem(t *testing.T) {
	const dt = 0.0001
	s := NewLambda()
	st := engine.State{LambdaEscape: 1.2}
	for range int(s.Tau / dt) {
		s.Update(dt, st)
	}
	// Após τ: 63,2 % do degrau de 0,2.
	near(t, "λ sonda após τ", s.Valor(), 1+0.2*(1-math.Exp(-1)), 1e-3)
}

func TestBancoSincronizaPorPasso(t *testing.T) {
	b := NewBank(NewLambda(), TPS{}, RPM{})
	defer b.Close()

	st := engine.State{TPS: 50, RPM: 4000, LambdaEscape: 1}
	for range 100 {
		v := b.Step(0.001, st)
		if len(v) != 3 {
			t.Fatalf("esperado 3 leituras, veio %d", len(v))
		}
		near(t, "lambda", v[0], 1.65, 1e-9)
		near(t, "tps", v[1], TPSVolts(50), 1e-9)
		near(t, "rpm", v[2], 1.65, 1e-9)
	}
}

// contador conta quantos passos recebeu; serve para conferir que nenhum
// sensor pula ou repete passo.
type contador struct{ n int }

func (c *contador) Name() string                         { return "contador" }
func (c *contador) Update(float64, engine.State) float64 { c.n++; return float64(c.n) }

func TestBancoNaoPulaPassos(t *testing.T) {
	cs := []*contador{{}, {}, {}, {}}
	b := NewBank(cs[0], cs[1], cs[2], cs[3])
	for i := 1; i <= 1000; i++ {
		for j, v := range b.Step(0.001, engine.State{}) {
			if int(v) != i {
				t.Fatalf("passo %d: sensor %d está no passo %v", i, j, v)
			}
		}
	}
	b.Close()
}
