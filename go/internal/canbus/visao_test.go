package canbus

import (
	"testing"

	"go.einride.tech/can"
)

func TestVisao(t *testing.T) {
	var v Visao
	if !v.Atualizar(EcuEstado{LambdaMedido: 1.05, LambdaAlvo: 1, TiUs: 2500, Modo: MalhaFechada}.Frame()) {
		t.Fatal("0x100 recusado")
	}
	v.Atualizar(PlantaEstado{LambdaSonda: 0.98, VazaoBicoPct: 80}.Frame())
	v.Atualizar(PlantaEstado{LambdaSonda: 0.99, VazaoBicoPct: 80}.Frame())
	if v.Atualizar(PainelPlanta{Acelerador: 10}.Frame()) {
		t.Error("frame do painel não deveria ser aceito")
	}
	if v.Atualizar(can.Frame{ID: IDEcuTempos, Length: 2}) {
		t.Error("frame curto aceito")
	}
	if v.NEstado != 1 || v.NPlanta != 2 || v.NTempos != 0 {
		t.Errorf("contagens %+v", v)
	}
	if v.Estado.TiUs != 2500 || v.Planta.LambdaSonda != 0.99 || v.Planta.VazaoBicoPct != 80 {
		t.Errorf("valores %+v", v)
	}
	if v.TempoPlanta() != 0.04 {
		t.Errorf("TempoPlanta %v", v.TempoPlanta())
	}
}
