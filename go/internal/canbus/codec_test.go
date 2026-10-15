package canbus

import (
	"math"
	"testing"

	"go.einride.tech/can"
)

func TestEcuEstadoIgualAoFirmware(t *testing.T) {
	// Mesmo vetor de firmware/tests/test_can_codec.c.
	m := EcuEstado{LambdaMedido: 1.0, LambdaAlvo: 0.95, TiUs: 4321, Correcao: -0.10, Modo: MalhaFechada, Contador: 3}
	f := m.Frame()
	want := [8]byte{0xE8, 0x03, 0xB6, 0x03, 0xE1, 0x10, 0xD8, 0x31}
	if f.Data != can.Data(want) || f.Length != 8 || f.ID != IDEcuEstado {
		t.Fatalf("frame = % X, esperado % X", f.Data, want)
	}
	got, err := DecodeEcuEstado(f)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.LambdaAlvo-0.95) > 1e-9 || got.TiUs != 4321 || math.Abs(got.Correcao+0.10) > 1e-9 ||
		got.Modo != MalhaFechada || got.Contador != 3 {
		t.Errorf("decodificado = %+v", got)
	}
}

func TestSensoresETempos(t *testing.T) {
	s, err := DecodeEcuSensores(can.Frame{ID: IDEcuSensores, Length: 4, Data: can.Data{0xD0, 0x07, 0x7B, 0x00}})
	if err != nil || s.RPM != 2000 || math.Abs(s.TPS-12.3) > 1e-9 {
		t.Errorf("sensores = %+v, %v", s, err)
	}
	tm, err := DecodeEcuTempos(can.Frame{ID: IDEcuTempos, Length: 8, Data: can.Data{0x10, 0x27, 0x0C, 0, 0x2C, 0x01, 2, 0}})
	if err != nil || tm != (EcuTempos{10000, 12, 300, 2}) {
		t.Errorf("tempos = %+v, %v", tm, err)
	}
}

func TestPainelEcu(t *testing.T) {
	f := PainelEcu{LambdaAlvo: 1.2, Modo: MalhaAberta, ZerarTempos: true}.Frame()
	if f.Length != 4 || f.Data[0] != 0xB0 || f.Data[1] != 0x04 || f.Data[2] != 0 || f.Data[3] != 1 {
		t.Errorf("frame = % X (o teste Robot usa B0040001)", f.Data[:4])
	}
}

func TestPlantaIdaEVolta(t *testing.T) {
	m := PlantaEstado{LambdaCilindro: 1.25, LambdaSonda: 1.1, AtrasoMs: 245, VazaoBicoPct: 80, VazamentoMg: 5}
	got, err := DecodePlantaEstado(m.Frame())
	if err != nil || math.Abs(got.LambdaCilindro-1.25) > 1e-9 || math.Abs(got.LambdaSonda-1.1) > 1e-9 ||
		got.AtrasoMs != 245 || got.VazaoBicoPct != 80 || got.VazamentoMg != 5 {
		t.Errorf("planta = %+v, %v", got, err)
	}
	p := PainelPlanta{Acelerador: 30, AjusteBico: -20, VazamentoMg: ManterVazamento}
	gp, err := DecodePainelPlanta(p.Frame())
	if err != nil || gp != p {
		t.Errorf("painel→planta = %+v, %v", gp, err)
	}
}

func TestRejeitaFrameErrado(t *testing.T) {
	if _, err := DecodeEcuEstado(can.Frame{ID: IDEcuEstado, Length: 4}); err == nil {
		t.Error("esperado erro com frame curto")
	}
	if _, err := DecodeEcuEstado(can.Frame{ID: IDEcuTempos, Length: 8}); err == nil {
		t.Error("esperado erro com ID errado")
	}
}

func TestSaturacao(t *testing.T) {
	f := EcuEstado{LambdaMedido: 100, LambdaAlvo: -1, Correcao: 1}.Frame()
	d, _ := DecodeEcuEstado(f)
	if d.LambdaMedido != 65.535 || d.LambdaAlvo != 0 || math.Abs(d.Correcao-0.3175) > 1e-9 {
		t.Errorf("saturação = %+v", d)
	}
}

func TestSensoresETemposIdaEVolta(t *testing.T) {
	s, err := DecodeEcuSensores(EcuSensores{RPM: 1996.4, TPS: 23.07}.Frame())
	if err != nil || s.RPM != 1996 || math.Abs(s.TPS-23.1) > 1e-9 {
		t.Errorf("sensores %+v %v", s, err)
	}
	in := EcuTempos{PeriodoUs: 10000, JitterMaxUs: 3, ExecMaxUs: 160, DeadlinesPerdidos: 1}
	if out, err := DecodeEcuTempos(in.Frame()); err != nil || out != in {
		t.Errorf("tempos %+v %v", out, err)
	}
}

func TestPainelEcuIdaEVolta(t *testing.T) {
	in := PainelEcu{LambdaAlvo: 1.05, Modo: MalhaAberta, ZerarTempos: true}
	out, err := DecodePainelEcu(in.Frame())
	if err != nil || math.Abs(out.LambdaAlvo-1.05) > 1e-9 || out.Modo != in.Modo || !out.ZerarTempos {
		t.Errorf("%+v %v", out, err)
	}
}
