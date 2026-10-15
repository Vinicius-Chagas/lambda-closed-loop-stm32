package renode

import (
	"fmt"
	"strconv"
	"strings"
)

// Sensores na ordem dos canais do ADC (ver renode/stm32f072_lambda.repl).
var SensorPeripherals = []string{"sysbus.adc.lambdaSensor", "sysbus.adc.tpsSensor", "sysbus.adc.rpmSensor"}

// Injector é o comando do bico lido nos registradores do timer.
type Injector struct {
	PulseUs  float64 // largura do pulso (CCR1, 1 µs por contagem)
	PeriodUs float64 // ciclo do motor (ARR + 1)
}

// ECU conduz o firmware no Renode em passos.
type ECU struct {
	mon *Monitor
}

// NewECU usa um monitor já conectado. Se script não for vazio, ele é
// carregado (ex.: "@/work/renode/lambda-vcan.resc").
func NewECU(mon *Monitor, script string) (*ECU, error) {
	if script != "" {
		if _, err := mon.Exec("include " + script); err != nil {
			return nil, err
		}
	}
	return &ECU{mon: mon}, nil
}

// Step aplica as tensões aos canais do ADC, roda dt segundos de tempo
// virtual e devolve o comando do injetor no fim do intervalo. Usa o comando
// `lockstep` (renode/lockstep.py, carregado pelo lambda.resc): uma ida e volta
// pelo TCP e um só comando do monitor por passo.
func (e *ECU) Step(volts []float64, dt float64) (Injector, error) {
	if len(volts) != len(SensorPeripherals) {
		return Injector{}, fmt.Errorf("renode: esperadas %d tensões, veio %d", len(SensorPeripherals), len(volts))
	}
	var b strings.Builder
	b.WriteString("lockstep")
	for _, v := range volts {
		b.WriteString(" " + strconv.FormatFloat(v, 'f', 4, 64))
	}
	b.WriteString(" " + strconv.FormatFloat(dt, 'f', -1, 64))

	lines, err := e.mon.Exec(b.String())
	if err != nil {
		return Injector{}, err
	}
	var fields []string
	for _, l := range lines {
		fields = append(fields, strings.Fields(l)...)
	}
	v, err := Hexes(fields)
	if err != nil {
		return Injector{}, err
	}
	if len(v) != 2 {
		return Injector{}, fmt.Errorf("renode: esperados CCR1 e ARR, saída %q", lines)
	}
	return Injector{PulseUs: float64(v[0]), PeriodUs: float64(v[1]) + 1}, nil
}
