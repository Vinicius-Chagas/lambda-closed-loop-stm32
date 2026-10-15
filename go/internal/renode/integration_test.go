//go:build renode

// Testes contra um Renode de verdade. Rodar com `make test-renode-go`, que
// sobe o container com o monitor em localhost:1234.
package renode

import (
	"math"
	"os"
	"sync"
	"testing"
	"time"
)

var (
	sharedOnce sync.Once
	sharedECU  *ECU
	sharedErr  error
)

// dialECU conecta e carrega o firmware uma única vez; os testes compartilham
// a mesma máquina (o Renode não permite recriar o hub CAN com o mesmo nome).
func dialECU(t *testing.T) *ECU {
	t.Helper()
	sharedOnce.Do(func() {
		addr := os.Getenv("RENODE_ADDR")
		if addr == "" {
			addr = "localhost:1234"
		}
		script := os.Getenv("RENODE_SCRIPT")
		if script == "" {
			script = "@renode/lambda.resc"
		}
		mon, err := Dial(addr, 60*time.Second)
		if err != nil {
			sharedErr = err
			return
		}
		sharedECU, sharedErr = NewECU(mon, script)
	})
	if sharedErr != nil {
		t.Fatal(sharedErr)
	}
	return sharedECU
}

func cicloEsperado(rpmVolts float64) float64 {
	counts := math.Round(rpmVolts / 3.3 * 4095)
	return math.Round(120e6 / (counts / 4095 * 8000))
}

func TestRenodeInjetorSegueRotacao(t *testing.T) {
	ecu := dialECU(t)

	var inj Injector
	var err error
	for range 60 { // 0,3 s em passos de 5 ms
		if inj, err = ecu.Step([]float64{1.65, 0.5, 3.0}, 0.005); err != nil {
			t.Fatal(err)
		}
	}
	if inj.PeriodUs != cicloEsperado(3.0) {
		t.Errorf("ciclo = %.0f µs, esperado %.0f", inj.PeriodUs, cicloEsperado(3.0))
	}
	if inj.PulseUs < 800 || inj.PulseUs > 60000 {
		t.Errorf("pulso fora da faixa: %.0f µs", inj.PulseUs)
	}

	for range 60 {
		if inj, err = ecu.Step([]float64{1.65, 0.3, 0.33}, 0.005); err != nil {
			t.Fatal(err)
		}
	}
	if inj.PeriodUs != cicloEsperado(0.33) {
		t.Errorf("ciclo em marcha lenta = %.0f µs, esperado %.0f", inj.PeriodUs, cicloEsperado(0.33))
	}
}

func TestRenodeVelocidadeDoLockstep(t *testing.T) {
	ecu := dialECU(t)
	const passos = 200
	start := time.Now()
	for range passos {
		if _, err := ecu.Step([]float64{1.65, 0.3, 0.33}, 0.005); err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start).Seconds()
	t.Logf("lockstep de 5 ms: %.0f passos/s, %.2fx tempo real", passos/elapsed, passos*0.005/elapsed)
}
