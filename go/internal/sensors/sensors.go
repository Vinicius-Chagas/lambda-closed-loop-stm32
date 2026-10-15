// Package sensors simula os sensores da bancada (docs/equacoes.md, seções 4 e 5).
//
// Cada sensor roda na sua própria goroutine, com estado próprio (a sonda tem
// dinâmica de 1ª ordem). O Bank funciona como uma barreira: a cada passo da
// simulação envia o estado do motor a todos os sensores e espera todas as
// leituras antes de seguir. O relógio é o da simulação, nunca o de parede.
package sensors

import (
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/engine"
)

// VRef é a tensão de referência do ADC da ECU.
const VRef = 3.3

// Sensor converte o estado do motor na tensão entregue ao ADC.
type Sensor interface {
	Name() string
	// Update avança dt segundos e devolve a tensão de saída.
	Update(dt float64, st engine.State) float64
}

// Lambda é a sonda banda larga: 1ª ordem sobre λ do escape e saída linear.
type Lambda struct {
	Tau    float64 // s
	lambda float64
}

// NewLambda cria a sonda já em λ = 1.
func NewLambda() *Lambda { return &Lambda{Tau: 0.100, lambda: 1} }

func (s *Lambda) Name() string { return "lambda" }

// Valor é o λ que a sonda está "vendo".
func (s *Lambda) Valor() float64 { return s.lambda }

func (s *Lambda) Update(dt float64, st engine.State) float64 {
	s.lambda += (st.LambdaEscape - s.lambda) * dt / s.Tau
	return LambdaVolts(s.lambda)
}

// LambdaVolts é a saída elétrica da sonda: 0,7..1,3 → 0..3,3 V, saturada.
func LambdaVolts(lambda float64) float64 {
	return clamp(VRef*(lambda-0.7)/0.6, 0, VRef)
}

// TPS é o potenciômetro da borboleta: 0..100 % → 0,3..3,0 V.
type TPS struct{}

func (TPS) Name() string { return "tps" }

func (TPS) Update(_ float64, st engine.State) float64 { return TPSVolts(st.TPS) }

// TPSVolts converte a abertura da borboleta em tensão.
func TPSVolts(tps float64) float64 { return 0.3 + 2.7*clamp(tps, 0, 100)/100 }

// RPM é o conversor frequência/tensão ligado à roda fônica: 0..8000 rpm → 0..3,3 V.
type RPM struct{}

func (RPM) Name() string { return "rpm" }

func (RPM) Update(_ float64, st engine.State) float64 { return RPMVolts(st.RPM) }

// RPMVolts converte a rotação em tensão.
func RPMVolts(rpm float64) float64 { return clamp(VRef*rpm/8000, 0, VRef) }

// Counts é a contagem que o ADC de 12 bits da ECU lê para uma tensão.
func Counts(v float64) int {
	return int(clamp(v, 0, VRef)/VRef*4095 + 0.5)
}

func clamp(x, lo, hi float64) float64 { return min(max(x, lo), hi) }
