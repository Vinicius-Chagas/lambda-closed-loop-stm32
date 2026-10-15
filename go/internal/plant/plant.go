// Package plant junta motor e sensores e conduz a ECU em passos (lockstep).
//
// A cada passo de sincronização a planta integra a física em subpassos de dt,
// entrega as tensões dos sensores à ECU, deixa a ECU rodar o mesmo intervalo
// de tempo e lê o novo comando do injetor. A ECU pode ser o firmware no Renode
// (internal/renode) ou o controle C compilado no host (internal/sil).
package plant

import (
	"fmt"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/engine"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/renode"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/sensors"
)

// ECU é quem calcula a injeção: recebe as tensões dos sensores (ordem dos
// canais do ADC), roda dt segundos e devolve o comando do injetor.
type ECU interface {
	Step(volts []float64, dt float64) (renode.Injector, error)
}

// Config define os passos de tempo.
type Config struct {
	Dt        float64 // passo da física, s (padrão 1 ms)
	SyncEvery int     // subpassos por sincronização com a ECU (padrão 5)
}

// Command é uma alteração pedida pelo painel. Campos nil ficam como estão.
type Command struct {
	TPS         *float64
	FatorBico   *float64
	VazamentoMg *float64
}

// Sample é o estado da bancada ao fim de um passo de sincronização.
type Sample struct {
	engine.State
	LambdaSonda float64
	Volts       []float64
	Injector    renode.Injector
	FatorBico   float64
	VazamentoMg float64
}

// Plant é a bancada simulada. Step deve ser chamado de uma só goroutine;
// Apply pode ser chamado de qualquer goroutine.
type Plant struct {
	cfg   Config
	eng   *engine.Engine
	sonda *sensors.Lambda
	bank  *sensors.Bank
	ecu   ECU
	inj   renode.Injector
	volts []float64
	cmds  chan Command
}

// New cria a planta em marcha lenta. O injetor começa com o pulso
// estequiométrico até a ECU responder.
func New(cfg Config, ecu ECU) *Plant {
	if cfg.Dt <= 0 {
		cfg.Dt = 0.001
	}
	if cfg.SyncEvery <= 0 {
		cfg.SyncEvery = 5
	}
	eng := engine.New(cfg.Dt)
	sonda := sensors.NewLambda()
	st := eng.State()
	p := &Plant{
		cfg:   cfg,
		eng:   eng,
		sonda: sonda,
		bank:  sensors.NewBank(sonda, sensors.TPS{}, sensors.RPM{}),
		ecu:   ecu,
		inj:   renode.Injector{PulseUs: engine.TiEstequiometrico(engine.MassaAr(st.TPS, st.RPM))},
		cmds:  make(chan Command, 16),
	}
	p.volts = []float64{sensors.LambdaVolts(1), sensors.TPSVolts(st.TPS), sensors.RPMVolts(st.RPM)}
	return p
}

// Close encerra as goroutines dos sensores.
func (p *Plant) Close() { p.bank.Close() }

// SyncPeriod é o intervalo de tempo simulado de cada Step.
func (p *Plant) SyncPeriod() float64 { return p.cfg.Dt * float64(p.cfg.SyncEvery) }

// Apply enfileira um comando; ele vale a partir do próximo Step.
func (p *Plant) Apply(c Command) {
	select {
	case p.cmds <- c:
	default: // fila cheia: descarta (o painel reenvia)
	}
}

func (p *Plant) drainCommands() {
	for {
		select {
		case c := <-p.cmds:
			if c.TPS != nil {
				p.eng.SetTPS(*c.TPS)
			}
			d := p.eng.Disturbances()
			if c.FatorBico != nil {
				d.FatorBico = *c.FatorBico
			}
			if c.VazamentoMg != nil {
				d.VazamentoMg = *c.VazamentoMg
			}
			p.eng.SetDisturbances(d)
		default:
			return
		}
	}
}

// Step avança um passo de sincronização.
func (p *Plant) Step() (Sample, error) {
	p.drainCommands()
	var st engine.State
	for range p.cfg.SyncEvery {
		st = p.eng.Step(p.inj.PulseUs)
		p.volts = p.bank.Step(p.cfg.Dt, st)
	}
	inj, err := p.ecu.Step(p.volts, p.SyncPeriod())
	if err != nil {
		return Sample{}, fmt.Errorf("planta: ECU: %w", err)
	}
	p.inj = inj
	d := p.eng.Disturbances()
	return Sample{
		State:       st,
		LambdaSonda: p.sonda.Valor(),
		Volts:       append([]float64(nil), p.volts...),
		Injector:    inj,
		FatorBico:   d.FatorBico,
		VazamentoMg: d.VazamentoMg,
	}, nil
}
