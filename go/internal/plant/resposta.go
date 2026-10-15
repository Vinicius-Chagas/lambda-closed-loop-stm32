package plant

import "math"

// Degrau resume a resposta de λ a uma perturbação em degrau.
type Degrau struct {
	DesvioMax   float64 // maior |λ − alvo| depois do degrau
	Sobressinal float64 // quanto passou do alvo no sentido oposto, em fração de DesvioMax
	Acomodacao  float64 // s, do degrau até entrar de vez na banda (0 se nunca saiu)
	Final       float64 // média de λ no último 0,5 s
	Acomodou    bool    // terminou dentro da banda
}

// AnalisarDegrau mede a resposta y(t) a um degrau aplicado em tDegrau.
// Sobressinal segue a definição usual: depois do maior desvio, o quanto λ
// passa do alvo no sentido contrário, dividido por esse desvio.
func AnalisarDegrau(t, y []float64, tDegrau, alvo, banda float64) Degrau {
	var d Degrau
	pico := -1
	ultimaFora := math.NaN()
	for i := range t {
		if t[i] < tDegrau {
			continue
		}
		e := math.Abs(y[i] - alvo)
		if e > d.DesvioMax {
			d.DesvioMax, pico = e, i
		}
		if e > banda {
			ultimaFora = t[i]
		}
	}
	if pico >= 0 {
		// maior excursão no sentido oposto ao pico, depois dele
		sinal := math.Copysign(1, y[pico]-alvo)
		passou := 0.0
		for i := pico; i < len(t); i++ {
			passou = math.Max(passou, -sinal*(y[i]-alvo))
		}
		d.Sobressinal = passou / d.DesvioMax
	}
	if !math.IsNaN(ultimaFora) {
		d.Acomodacao = ultimaFora - tDegrau
	}
	n, soma := 0, 0.0
	for i := range t {
		if t[i] >= t[len(t)-1]-0.5 {
			soma += y[i]
			n++
		}
	}
	if n > 0 {
		d.Final = soma / float64(n)
	}
	d.Acomodou = len(t) > 0 && math.Abs(y[len(y)-1]-alvo) <= banda
	return d
}
