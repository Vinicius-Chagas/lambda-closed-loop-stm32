// Package canbus define os frames CAN da bancada (can/lambda.dbc) e o acesso
// ao SocketCAN. O codec é o mesmo de firmware/control/can_codec.c.
package canbus

import (
	"encoding/binary"
	"fmt"
	"math"

	"go.einride.tech/can"
)

// IDs de 11 bits.
const (
	IDEcuEstado    = 0x100
	IDEcuSensores  = 0x101
	IDEcuTempos    = 0x110
	IDPainelEcu    = 0x200
	IDPlanta       = 0x300
	IDPainelPlanta = 0x310
)

// Valores especiais dos comandos.
const (
	ManterModo       = 0xFF
	ManterAcelerador = 0xFF
	ManterBico       = math.MinInt8
	ManterVazamento  = 0xFF
)

// Modo da malha de controle.
const (
	MalhaAberta  = 0
	MalhaFechada = 1
)

// EcuEstado é o frame 0x100.
type EcuEstado struct {
	LambdaMedido float64
	LambdaAlvo   float64
	TiUs         uint16
	Correcao     float64 // fração (-0,30..0,30)
	Modo         uint8
	Contador     uint8
}

// EcuSensores é o frame 0x101.
type EcuSensores struct {
	RPM float64
	TPS float64 // %
}

// EcuTempos é o frame 0x110.
type EcuTempos struct {
	PeriodoUs         uint16
	JitterMaxUs       uint16
	ExecMaxUs         uint16
	DeadlinesPerdidos uint16
}

// PainelEcu é o frame 0x200.
type PainelEcu struct {
	LambdaAlvo  float64 // 0 = mantém
	Modo        uint8   // MalhaAberta, MalhaFechada ou ManterModo
	ZerarTempos bool
}

// PlantaEstado é o frame 0x300.
type PlantaEstado struct {
	LambdaCilindro float64
	LambdaSonda    float64
	AtrasoMs       uint16
	VazaoBicoPct   uint8 // 100 = nominal
	VazamentoMg    uint8
}

// PainelPlanta é o frame 0x310.
type PainelPlanta struct {
	Acelerador  uint8 // %, ou ManterAcelerador
	AjusteBico  int8  // %, ou ManterBico
	VazamentoMg uint8 // ou ManterVazamento
}

func u16(x, res float64) uint16 {
	v := math.Round(x / res)
	return uint16(min(max(v, 0), math.MaxUint16))
}

func frame(id uint32, n uint8) can.Frame { return can.Frame{ID: id, Length: n} }

func check(f can.Frame, id uint32, n uint8) error {
	if f.ID != id || f.IsExtended || f.IsRemote {
		return fmt.Errorf("canbus: esperado frame 0x%X, veio 0x%X", id, f.ID)
	}
	if f.Length < n {
		return fmt.Errorf("canbus: frame 0x%X com %d bytes, esperado %d", id, f.Length, n)
	}
	return nil
}

// Frame codifica o estado da ECU.
func (m EcuEstado) Frame() can.Frame {
	f := frame(IDEcuEstado, 8)
	binary.LittleEndian.PutUint16(f.Data[0:], u16(m.LambdaMedido, 0.001))
	binary.LittleEndian.PutUint16(f.Data[2:], u16(m.LambdaAlvo, 0.001))
	binary.LittleEndian.PutUint16(f.Data[4:], m.TiUs)
	f.Data[6] = byte(int8(min(max(math.Round(m.Correcao/0.0025), -127), 127)))
	f.Data[7] = m.Modo&1 | m.Contador<<4
	return f
}

// DecodeEcuEstado lê o frame 0x100.
func DecodeEcuEstado(f can.Frame) (EcuEstado, error) {
	if err := check(f, IDEcuEstado, 8); err != nil {
		return EcuEstado{}, err
	}
	return EcuEstado{
		LambdaMedido: float64(binary.LittleEndian.Uint16(f.Data[0:])) * 0.001,
		LambdaAlvo:   float64(binary.LittleEndian.Uint16(f.Data[2:])) * 0.001,
		TiUs:         binary.LittleEndian.Uint16(f.Data[4:]),
		Correcao:     float64(int8(f.Data[6])) * 0.0025,
		Modo:         f.Data[7] & 1,
		Contador:     f.Data[7] >> 4,
	}, nil
}

// DecodeEcuSensores lê o frame 0x101.
func DecodeEcuSensores(f can.Frame) (EcuSensores, error) {
	if err := check(f, IDEcuSensores, 4); err != nil {
		return EcuSensores{}, err
	}
	return EcuSensores{
		RPM: float64(binary.LittleEndian.Uint16(f.Data[0:])),
		TPS: float64(binary.LittleEndian.Uint16(f.Data[2:])) * 0.1,
	}, nil
}

// DecodeEcuTempos lê o frame 0x110.
func DecodeEcuTempos(f can.Frame) (EcuTempos, error) {
	if err := check(f, IDEcuTempos, 8); err != nil {
		return EcuTempos{}, err
	}
	return EcuTempos{
		PeriodoUs:         binary.LittleEndian.Uint16(f.Data[0:]),
		JitterMaxUs:       binary.LittleEndian.Uint16(f.Data[2:]),
		ExecMaxUs:         binary.LittleEndian.Uint16(f.Data[4:]),
		DeadlinesPerdidos: binary.LittleEndian.Uint16(f.Data[6:]),
	}, nil
}

// Frame codifica o comando do painel para a ECU.
func (m PainelEcu) Frame() can.Frame {
	f := frame(IDPainelEcu, 4)
	binary.LittleEndian.PutUint16(f.Data[0:], u16(m.LambdaAlvo, 0.001))
	f.Data[2] = m.Modo
	if m.ZerarTempos {
		f.Data[3] = 1
	}
	return f
}

// Frame codifica o estado da planta.
func (m PlantaEstado) Frame() can.Frame {
	f := frame(IDPlanta, 8)
	binary.LittleEndian.PutUint16(f.Data[0:], u16(m.LambdaCilindro, 0.001))
	binary.LittleEndian.PutUint16(f.Data[2:], u16(m.LambdaSonda, 0.001))
	binary.LittleEndian.PutUint16(f.Data[4:], m.AtrasoMs)
	f.Data[6] = m.VazaoBicoPct
	f.Data[7] = m.VazamentoMg
	return f
}

// DecodePlantaEstado lê o frame 0x300.
func DecodePlantaEstado(f can.Frame) (PlantaEstado, error) {
	if err := check(f, IDPlanta, 8); err != nil {
		return PlantaEstado{}, err
	}
	return PlantaEstado{
		LambdaCilindro: float64(binary.LittleEndian.Uint16(f.Data[0:])) * 0.001,
		LambdaSonda:    float64(binary.LittleEndian.Uint16(f.Data[2:])) * 0.001,
		AtrasoMs:       binary.LittleEndian.Uint16(f.Data[4:]),
		VazaoBicoPct:   f.Data[6],
		VazamentoMg:    f.Data[7],
	}, nil
}

// Frame codifica o comando do painel para a planta.
func (m PainelPlanta) Frame() can.Frame {
	f := frame(IDPainelPlanta, 3)
	f.Data[0] = m.Acelerador
	f.Data[1] = byte(m.AjusteBico)
	f.Data[2] = m.VazamentoMg
	return f
}

// DecodePainelPlanta lê o frame 0x310.
func DecodePainelPlanta(f can.Frame) (PainelPlanta, error) {
	if err := check(f, IDPainelPlanta, 3); err != nil {
		return PainelPlanta{}, err
	}
	return PainelPlanta{Acelerador: f.Data[0], AjusteBico: int8(f.Data[1]), VazamentoMg: f.Data[2]}, nil
}
