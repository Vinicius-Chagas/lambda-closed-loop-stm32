package sensors

import (
	"sync"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/engine"
)

type tick struct {
	dt float64
	st engine.State
}

// Bank roda um conjunto de sensores em paralelo, sincronizados por passo.
type Bank struct {
	sensors []Sensor
	in      []chan tick
	out     []float64
	wg      sync.WaitGroup // barreira do passo
	done    sync.WaitGroup // goroutines vivas
}

// NewBank inicia uma goroutine por sensor. A ordem das leituras em Step é a
// ordem dos sensores aqui (= ordem dos canais do ADC).
func NewBank(sensors ...Sensor) *Bank {
	b := &Bank{
		sensors: sensors,
		in:      make([]chan tick, len(sensors)),
		out:     make([]float64, len(sensors)),
	}
	for i, s := range sensors {
		b.in[i] = make(chan tick)
		b.done.Add(1)
		go b.run(i, s)
	}
	return b
}

func (b *Bank) run(i int, s Sensor) {
	defer b.done.Done()
	for t := range b.in[i] {
		b.out[i] = s.Update(t.dt, t.st) // cada goroutine escreve só no seu índice
		b.wg.Done()
	}
}

// Step entrega o estado do motor a todos os sensores e devolve as tensões
// depois que todos terminaram o passo.
func (b *Bank) Step(dt float64, st engine.State) []float64 {
	b.wg.Add(len(b.sensors))
	for _, c := range b.in {
		c <- tick{dt: dt, st: st}
	}
	b.wg.Wait()
	return append([]float64(nil), b.out...)
}

// Close encerra as goroutines.
func (b *Bank) Close() {
	for _, c := range b.in {
		close(c)
	}
	b.done.Wait()
}
