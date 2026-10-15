package main

import (
	"testing"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/canbus"
)

func TestCommandFrom(t *testing.T) {
	c := commandFrom(canbus.PainelPlanta{Acelerador: 30, AjusteBico: -20, VazamentoMg: canbus.ManterVazamento})
	if c.TPS == nil || *c.TPS != 30 || c.FatorBico == nil || *c.FatorBico != 0.8 || c.VazamentoMg != nil {
		t.Errorf("comando = %+v", c)
	}
	c = commandFrom(canbus.PainelPlanta{Acelerador: canbus.ManterAcelerador, AjusteBico: 90, VazamentoMg: 7})
	if c.TPS != nil || *c.FatorBico != 1.5 || *c.VazamentoMg != 7 {
		t.Errorf("comando = %+v", c)
	}
}
