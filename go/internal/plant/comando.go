package plant

import "github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/canbus"

// CommandFromCAN converte o frame 0x310 do painel num comando da planta.
// O ajuste do bico fica limitado a ±50 %.
func CommandFromCAN(m canbus.PainelPlanta) Command {
	var c Command
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
