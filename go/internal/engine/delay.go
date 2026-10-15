package engine

// DelayLine guarda amostras em intervalos fixos de tempo e devolve o valor
// de um instante no passado (atraso de transporte do escape).
type DelayLine struct {
	dt   float64
	buf  []float64
	head int // posição da amostra mais recente
}

// NewDelayLine cria uma linha para atrasos de até maxDelay segundos, com
// amostras a cada dt, toda preenchida com initial.
func NewDelayLine(dt, maxDelay, initial float64) *DelayLine {
	n := int(maxDelay/dt) + 2
	d := &DelayLine{dt: dt, buf: make([]float64, n)}
	for i := range d.buf {
		d.buf[i] = initial
	}
	return d
}

// Push grava a amostra do instante atual.
func (d *DelayLine) Push(v float64) {
	d.head = (d.head + 1) % len(d.buf)
	d.buf[d.head] = v
}

// At devolve o valor de delay segundos atrás, interpolando entre amostras.
func (d *DelayLine) At(delay float64) float64 {
	pos := delay / d.dt
	maxPos := float64(len(d.buf) - 2)
	pos = clamp(pos, 0, maxPos)
	i := int(pos)
	frac := pos - float64(i)
	a := d.buf[d.index(i)]
	b := d.buf[d.index(i+1)]
	return a + (b-a)*frac
}

func (d *DelayLine) index(back int) int {
	n := len(d.buf)
	return ((d.head-back)%n + n) % n
}
