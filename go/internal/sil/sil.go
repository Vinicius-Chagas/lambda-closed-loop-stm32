// Package sil roda o controle C do firmware (firmware/control, via cgo) no
// host, como ECU da planta: software-in-the-loop, sem o Renode. Serve para
// testar a malha fechada (tabela + PI) contra o modelo do motor em segundos.
package sil

/*
#cgo CFLAGS: -std=c11 -I${SRCDIR}/../../../firmware/control
#cgo LDFLAGS: -lm
#include "ecu.h"
*/
import "C"

import (
	"fmt"
	"math"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/renode"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/sensors"
)

// PeriodoControle é o período da ControlTask (um bloco do DMA do ADC).
const PeriodoControle = 0.010

// Modos da malha (can/lambda.dbc).
const (
	MalhaAberta  = 0
	MalhaFechada = 1
)

// Saida é o resultado do último passo de controle.
type Saida struct {
	LambdaMed, TPS, RPM float64
	TiUs, CicloUs       float64
	TBaseUs, Correcao   float64
}

// ECU implementa plant.ECU com o código C. Como no firmware, as amostras de
// cada período de 10 ms são somadas e o controle roda com a média.
type ECU struct {
	e      C.ecu_t
	soma   [3]float64
	tempo  float64 // tempo acumulado no bloco atual
	inj    renode.Injector
	saida  Saida
	passos int
}

// New cria a ECU no estado de partida do firmware (λ alvo 1,0, malha fechada).
func New() *ECU {
	ecu := &ECU{}
	C.ecu_init(&ecu.e)
	return ecu
}

// SetAlvo e SetModo são os comandos do painel (0x200); valores inválidos são ignorados.
func (ecu *ECU) SetAlvo(lambda float64) { C.ecu_set_alvo(&ecu.e, C.float(lambda)) }
func (ecu *ECU) SetModo(modo int)       { C.ecu_set_modo(&ecu.e, C.modo_t(modo)) }

// Saida devolve o resultado do último período de controle.
func (ecu *ECU) Saida() Saida { return ecu.saida }

// Passos conta quantas vezes o controle rodou.
func (ecu *ECU) Passos() int { return ecu.passos }

// Step recebe as tensões dos canais (λ, TPS, RPM) válidas durante dt.
func (ecu *ECU) Step(volts []float64, dt float64) (renode.Injector, error) {
	if len(volts) != 3 {
		return renode.Injector{}, fmt.Errorf("sil: esperava 3 tensões, veio %d", len(volts))
	}
	if dt <= 0 || dt > PeriodoControle+1e-9 {
		return renode.Injector{}, fmt.Errorf("sil: dt %.4f s fora de (0, %.3f]", dt, PeriodoControle)
	}
	for i, v := range volts {
		ecu.soma[i] += float64(sensors.Counts(v)) * dt
	}
	ecu.tempo += dt
	if ecu.tempo >= PeriodoControle-1e-9 {
		ecu.controlar()
	}
	return ecu.inj, nil
}

func (ecu *ECU) controlar() {
	var adc [3]C.uint16_t
	for i := range adc {
		adc[i] = C.uint16_t(math.Round(ecu.soma[i] / ecu.tempo))
		ecu.soma[i] = 0
	}
	ecu.tempo = 0

	var out C.ecu_out_t
	C.ecu_step(&ecu.e, &adc[0], &out)
	ecu.passos++
	ecu.saida = Saida{
		LambdaMed: float64(out.lambda_med),
		TPS:       float64(out.tps),
		RPM:       float64(out.rpm),
		TiUs:      float64(out.ctrl.ti_us),
		CicloUs:   float64(out.ctrl.ciclo_us),
		TBaseUs:   float64(out.ctrl.t_base_us),
		Correcao:  float64(out.ctrl.correcao),
	}
	ecu.inj = renode.Injector{PulseUs: ecu.saida.TiUs, PeriodUs: ecu.saida.CicloUs}
}
