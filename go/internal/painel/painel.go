// Package painel é a lógica do painel da bancada, sem a parte gráfica:
// guarda o que chega pelo CAN, o histórico do gráfico e transforma teclas em
// frames de comando. O desenho fica em cmd/painel (raylib).
package painel

import (
	"math"
	"time"

	"go.einride.tech/can"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/canbus"
)

// Valores das perturbações das teclas 1 e 2.
const (
	PerturbacaoBico      = -20 // % de vazão do bico (bico "entupido")
	PerturbacaoVazamento = 10  // mg de ar falso por ciclo
	PassoAcelerador      = 5   // % por toque nas setas
	Janela               = 10.0
	BandaIdeal           = 0.02
	SemSinal             = 2 * time.Second
)

// Amostra é um ponto do gráfico (um frame 0x300).
type Amostra struct {
	T              float64 // tempo simulado, s
	LambdaCilindro float64
	LambdaSonda    float64
	LambdaAlvo     float64
}

// Tecla é uma ação do painel.
type Tecla int

const (
	AceleradorMais Tecla = iota
	AceleradorMenos
	AlternarBico
	AlternarVazamento
	AlternarMalha
)

// Estado é tudo o que o painel mostra.
type Estado struct {
	canbus.Visao
	hist   []Amostra
	ultimo time.Time
}

// Atualizar processa um frame recebido em agora.
func (e *Estado) Atualizar(f can.Frame, agora time.Time) {
	if !e.Visao.Atualizar(f) {
		return
	}
	e.ultimo = agora
	if f.ID != canbus.IDPlanta {
		return
	}
	t := e.TempoPlanta()
	e.hist = append(e.hist, Amostra{
		T: t, LambdaCilindro: e.Planta.LambdaCilindro, LambdaSonda: e.Planta.LambdaSonda,
		LambdaAlvo: e.Estado.LambdaAlvo,
	})
	// descarta o que saiu da janela (com folga, para não copiar a cada frame)
	if len(e.hist) > 0 && t-e.hist[0].T > 2*Janela {
		i := 0
		for i < len(e.hist) && t-e.hist[i].T > Janela {
			i++
		}
		e.hist = append(e.hist[:0], e.hist[i:]...)
	}
}

// Historico devolve as amostras dos últimos Janela segundos (simulados).
func (e *Estado) Historico() []Amostra {
	if len(e.hist) == 0 {
		return nil
	}
	fim := e.hist[len(e.hist)-1].T
	i := len(e.hist)
	for i > 0 && fim-e.hist[i-1].T <= Janela {
		i--
	}
	return e.hist[i:]
}

// Conectado diz se chegou algum frame conhecido há pouco.
func (e *Estado) Conectado(agora time.Time) bool {
	return !e.ultimo.IsZero() && agora.Sub(e.ultimo) < SemSinal
}

// Comando devolve o frame que a tecla deve enviar. O ponto de partida é o
// que a bancada informa no CAN, então o painel pode reiniciar a qualquer hora.
func (e *Estado) Comando(t Tecla) can.Frame {
	manter := canbus.PainelPlanta{
		Acelerador: canbus.ManterAcelerador, AjusteBico: canbus.ManterBico, VazamentoMg: canbus.ManterVazamento,
	}
	switch t {
	case AceleradorMais, AceleradorMenos:
		passo := PassoAcelerador
		if t == AceleradorMenos {
			passo = -passo
		}
		atual := int(math.Round(e.Sensores.TPS/PassoAcelerador)) * PassoAcelerador
		manter.Acelerador = uint8(min(max(atual+passo, 0), 100))
	case AlternarBico:
		manter.AjusteBico = PerturbacaoBico
		if e.BicoPerturbado() {
			manter.AjusteBico = 0
		}
	case AlternarVazamento:
		manter.VazamentoMg = PerturbacaoVazamento
		if e.VazamentoAtivo() {
			manter.VazamentoMg = 0
		}
	case AlternarMalha:
		modo := uint8(canbus.MalhaAberta)
		if e.Estado.Modo == canbus.MalhaAberta {
			modo = canbus.MalhaFechada
		}
		return canbus.PainelEcu{Modo: modo}.Frame()
	}
	return manter.Frame()
}

// BicoPerturbado e VazamentoAtivo dizem se as perturbações estão ligadas.
func (e *Estado) BicoPerturbado() bool { return e.NPlanta > 0 && e.Planta.VazaoBicoPct != 100 }
func (e *Estado) VazamentoAtivo() bool { return e.Planta.VazamentoMg > 0 }

// Mistura é a classificação mostrada ao público.
type Mistura int

const (
	Rica Mistura = iota
	Ideal
	Pobre
)

func (m Mistura) String() string {
	return [...]string{"RICA", "IDEAL", "POBRE"}[m]
}

// Classificar compara λ com o alvo (±BandaIdeal é ideal).
func Classificar(lambda, alvo float64) Mistura {
	switch {
	case lambda < alvo-BandaIdeal:
		return Rica
	case lambda > alvo+BandaIdeal:
		return Pobre
	}
	return Ideal
}

// Escala da barra de mistura.
const (
	BarraMin = 0.80
	BarraMax = 1.20
)

// PosicaoBarra é a posição de λ na barra, de 0 (rica) a 1 (pobre).
func PosicaoBarra(lambda float64) float64 {
	return min(max((lambda-BarraMin)/(BarraMax-BarraMin), 0), 1)
}
