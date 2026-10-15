// Planta: simula motor, escape e sensores e conduz a ECU no Renode.
//
//	planta -renode localhost:1234 -can vcan0 -csv corrida.csv
//
// Recebe comandos do painel (0x310) e publica o estado da planta (0x300) no CAN.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/canbus"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/plant"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/renode"
)

func main() {
	addr := flag.String("renode", "localhost:1234", "endereço do monitor do Renode")
	script := flag.String("script", "@renode/lambda-vcan.resc", "script carregado no Renode (vazio = já carregado)")
	iface := flag.String("can", "vcan0", "interface SocketCAN (vazio = sem CAN)")
	dt := flag.Float64("dt", 0.001, "passo da física, s")
	sync := flag.Int("sync", 5, "subpassos por sincronização com a ECU")
	csvPath := flag.String("csv", "", "grava as amostras neste arquivo CSV")
	duration := flag.Float64("duracao", 0, "tempo simulado, s (0 = sem fim)")
	realtime := flag.Bool("tempo-real", true, "não deixa a simulação andar mais rápido que o relógio")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *addr, *script, *iface, *dt, *sync, *csvPath, *duration, *realtime); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, addr, script, iface string, dt float64, sync int, csvPath string,
	duration float64, realtime bool) error {
	mon, err := renode.Dial(addr, 60*time.Second)
	if err != nil {
		return err
	}
	defer mon.Close()
	ecu, err := renode.NewECU(mon, script)
	if err != nil {
		return err
	}

	p := plant.New(plant.Config{Dt: dt, SyncEvery: sync}, ecu)
	defer p.Close()

	var csvw *plant.CSVWriter
	if csvPath != "" {
		f, err := os.Create(csvPath)
		if err != nil {
			return err
		}
		defer f.Close()
		if csvw, err = plant.NewCSVWriter(f); err != nil {
			return err
		}
		defer csvw.Flush()
	}

	var bus *canbus.Bus
	if iface != "" {
		if bus, err = canbus.Open(ctx, iface); err != nil {
			return fmt.Errorf("abrindo %s: %w (rode scripts/setup-vcan.sh)", iface, err)
		}
		defer bus.Close()
		go receiveCommands(bus, p)
	}

	log.Printf("planta: lockstep de %.0f ms com o Renode em %s", p.SyncPeriod()*1000, addr)
	start := time.Now()
	nextCAN, nextLog := 0.0, 0.0
	for ctx.Err() == nil {
		s, err := p.Step()
		if err != nil {
			return err
		}
		if csvw != nil {
			if err := csvw.Write(s); err != nil {
				return err
			}
		}
		if bus != nil && s.Tempo >= nextCAN {
			nextCAN += 0.020
			if err := bus.Send(ctx, plantaFrame(s).Frame()); err != nil && ctx.Err() == nil {
				log.Printf("CAN: %v", err)
			}
		}
		if s.Tempo >= nextLog {
			nextLog += 1
			log.Printf("t=%5.1fs TPS=%3.0f%% RPM=%4.0f λcil=%.3f λsonda=%.3f ti=%5.0fus (%.2fx tempo real)",
				s.Tempo, s.TPS, s.RPM, s.LambdaCil, s.LambdaSonda, s.Injector.PulseUs,
				s.Tempo/time.Since(start).Seconds())
		}
		if duration > 0 && s.Tempo >= duration {
			break
		}
		if realtime {
			if ahead := time.Duration(s.Tempo*float64(time.Second)) - time.Since(start); ahead > 0 {
				time.Sleep(ahead)
			}
		}
	}
	return nil
}

func plantaFrame(s plant.Sample) canbus.PlantaEstado {
	return canbus.PlantaEstado{
		LambdaCilindro: s.LambdaCil,
		LambdaSonda:    s.LambdaSonda,
		AtrasoMs:       uint16(s.Atraso*1000 + 0.5),
		VazaoBicoPct:   uint8(min(max(s.FatorBico*100+0.5, 0), 255)),
		VazamentoMg:    uint8(min(max(s.VazamentoMg+0.5, 0), 255)),
	}
}

func receiveCommands(bus *canbus.Bus, p *plant.Plant) {
	for {
		f, ok := bus.Receive()
		if !ok {
			return
		}
		if f.ID != canbus.IDPainelPlanta {
			continue
		}
		m, err := canbus.DecodePainelPlanta(f)
		if err != nil {
			continue
		}
		p.Apply(commandFrom(m))
	}
}

func commandFrom(m canbus.PainelPlanta) plant.Command {
	var c plant.Command
	if m.Acelerador != canbus.ManterAcelerador {
		v := float64(min(m.Acelerador, 100))
		c.TPS = &v
	}
	if m.AjusteBico != canbus.ManterBico {
		v := 1 + float64(max(min(m.AjusteBico, 50), -50))/100
		c.FatorBico = &v
	}
	if m.VazamentoMg != canbus.ManterVazamento {
		v := float64(m.VazamentoMg)
		c.VazamentoMg = &v
	}
	return c
}
