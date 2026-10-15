package main

import (
	"context"
	"time"

	"go.einride.tech/can"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/canbus"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/plant"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/sil"
)

// fonte entrega os frames da bancada e recebe os comandos do painel.
type fonte interface {
	Frames() <-chan can.Frame
	Enviar(can.Frame) error
	Close()
}

// fonteCAN é a bancada de verdade, pelo SocketCAN.
type fonteCAN struct {
	bus    *canbus.Bus
	frames chan can.Frame
}

func abrirCAN(iface string) (*fonteCAN, error) {
	bus, err := canbus.Open(context.Background(), iface)
	if err != nil {
		return nil, err
	}
	f := &fonteCAN{bus: bus, frames: make(chan can.Frame, 1024)}
	go func() {
		defer close(f.frames)
		for {
			fr, ok := bus.Receive()
			if !ok {
				return
			}
			select {
			case f.frames <- fr:
			default: // tela parada: descarta, o próximo frame traz o estado atual
			}
		}
	}()
	return f, nil
}

func (f *fonteCAN) Frames() <-chan can.Frame { return f.frames }
func (f *fonteCAN) Close()                   { f.bus.Close() }

func (f *fonteCAN) Enviar(fr can.Frame) error {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	return f.bus.Send(ctx, fr)
}

// fonteDemo roda a planta com o controle C (SIL) dentro do painel, em tempo
// real, e produz os mesmos frames da bancada. Serve para testar o painel sem
// Renode nem vcan0 (e para a captura de tela da CI).
type fonteDemo struct {
	frames chan can.Frame
	cmds   chan can.Frame
	parar  chan struct{}
}

func abrirDemo() *fonteDemo {
	d := &fonteDemo{
		frames: make(chan can.Frame, 1024),
		cmds:   make(chan can.Frame, 16),
		parar:  make(chan struct{}),
	}
	go d.rodar()
	return d
}

func (d *fonteDemo) Frames() <-chan can.Frame { return d.frames }
func (d *fonteDemo) Close()                   { close(d.parar) }

func (d *fonteDemo) Enviar(fr can.Frame) error {
	d.cmds <- fr
	return nil
}

func (d *fonteDemo) rodar() {
	ecu := sil.New()
	p := plant.New(plant.Config{}, ecu)
	defer p.Close()
	tps := 23.0
	p.Apply(plant.Command{TPS: &tps})

	tick := time.NewTicker(time.Duration(canbus.PeriodoPlanta * float64(time.Second)))
	defer tick.Stop()
	var contador uint8
	for n := 0; ; n++ {
		select {
		case <-d.parar:
			return
		case fr := <-d.cmds:
			d.comando(fr, ecu, p)
		case <-tick.C:
			var s plant.Sample
			for range int(canbus.PeriodoPlanta / p.SyncPeriod()) {
				var err error
				if s, err = p.Step(); err != nil {
					return
				}
			}
			o := ecu.Saida()
			contador++
			d.publicar(canbus.EcuEstado{
				LambdaMedido: o.LambdaMed, LambdaAlvo: ecu.Alvo(), TiUs: uint16(min(o.TiUs, 65535)),
				Correcao: o.Correcao, Modo: uint8(ecu.Modo()), Contador: contador & 0x0F,
			}.Frame())
			d.publicar(canbus.EcuSensores{RPM: o.RPM, TPS: o.TPS}.Frame())
			d.publicar(canbus.PlantaEstado{
				LambdaCilindro: s.LambdaCil, LambdaSonda: s.LambdaSonda, AtrasoMs: uint16(s.Atraso*1000 + 0.5),
				VazaoBicoPct: uint8(s.FatorBico*100 + 0.5), VazamentoMg: uint8(s.VazamentoMg + 0.5),
			}.Frame())
		}
	}
}

func (d *fonteDemo) comando(fr can.Frame, ecu *sil.ECU, p *plant.Plant) {
	switch fr.ID {
	case canbus.IDPainelPlanta:
		if m, err := canbus.DecodePainelPlanta(fr); err == nil {
			p.Apply(plant.CommandFromCAN(m))
		}
	case canbus.IDPainelEcu:
		if m, err := canbus.DecodePainelEcu(fr); err == nil {
			if m.LambdaAlvo != 0 {
				ecu.SetAlvo(m.LambdaAlvo)
			}
			if m.Modo != canbus.ManterModo {
				ecu.SetModo(int(m.Modo))
			}
		}
	}
}

func (d *fonteDemo) publicar(fr can.Frame) {
	select {
	case d.frames <- fr:
	default:
	}
}
