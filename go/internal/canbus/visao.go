package canbus

import "go.einride.tech/can"

// Visao guarda o último valor de cada frame visto no barramento e quantos
// chegaram. É o que o painel mostra e o que o teste e2e confere.
type Visao struct {
	Estado   EcuEstado
	Sensores EcuSensores
	Tempos   EcuTempos
	Planta   PlantaEstado

	NEstado, NSensores, NTempos, NPlanta int
}

// Atualizar decodifica f. Devolve false para frames de outros IDs ou inválidos.
func (v *Visao) Atualizar(f can.Frame) bool {
	var err error
	switch f.ID {
	case IDEcuEstado:
		if v.Estado, err = DecodeEcuEstado(f); err == nil {
			v.NEstado++
		}
	case IDEcuSensores:
		if v.Sensores, err = DecodeEcuSensores(f); err == nil {
			v.NSensores++
		}
	case IDEcuTempos:
		if v.Tempos, err = DecodeEcuTempos(f); err == nil {
			v.NTempos++
		}
	case IDPlanta:
		if v.Planta, err = DecodePlantaEstado(f); err == nil {
			v.NPlanta++
		}
	default:
		return false
	}
	return err == nil
}

// TempoPlanta é o tempo simulado estimado pelos frames 0x300 (um a cada 20 ms).
func (v *Visao) TempoPlanta() float64 { return float64(v.NPlanta) * PeriodoPlanta }

// PeriodoPlanta é o intervalo, em tempo simulado, entre frames 0x300.
const PeriodoPlanta = 0.020
