package engine

// Disturbances são as perturbações que o painel pode aplicar.
type Disturbances struct {
	FatorBico   float64 // vazão real / nominal (1 = bico novo; 0,8 = entrega 20 % a menos)
	VazamentoMg float64 // ar não medido por cilindro e ciclo
}

// State é o estado do motor num instante.
type State struct {
	Tempo        float64 // s desde o início
	TPS          float64 // %
	RPM          float64
	MassaArMg    float64 // ar real no cilindro (com vazamento)
	MassaCombMg  float64
	LambdaCil    float64 // λ da mistura queimada agora
	LambdaEscape float64 // λ do gás que está chegando à sonda
	Atraso       float64 // s, atraso de transporte atual
}

// Engine integra o modelo no tempo. Não é seguro para uso concorrente.
type Engine struct {
	dt    float64
	st    State
	dist  Disturbances
	delay *DelayLine
}

// New cria o motor em marcha lenta, com mistura estequiométrica no escape.
// dt é o passo de integração em segundos.
func New(dt float64) *Engine {
	e := &Engine{
		dt:    dt,
		dist:  Disturbances{FatorBico: 1},
		delay: NewDelayLine(dt, AtrasoTransporte(300)+0.1, 1.0),
	}
	e.st.RPM = RPMLenta
	e.st.LambdaCil = 1
	e.st.LambdaEscape = 1
	e.st.Atraso = AtrasoTransporte(RPMLenta)
	return e
}

// Dt é o passo de integração.
func (e *Engine) Dt() float64 { return e.dt }

// State devolve o estado atual.
func (e *Engine) State() State { return e.st }

// SetTPS ajusta a abertura da borboleta (0..100 %).
func (e *Engine) SetTPS(tps float64) { e.st.TPS = clamp(tps, 0, 100) }

// Disturbances devolve as perturbações ativas.
func (e *Engine) Disturbances() Disturbances { return e.dist }

// SetDisturbances troca as perturbações ativas.
func (e *Engine) SetDisturbances(d Disturbances) {
	d.FatorBico = clamp(d.FatorBico, 0, 2)
	d.VazamentoMg = max(d.VazamentoMg, 0)
	e.dist = d
}

// Step avança um passo dt com o pulso de injeção tiUs (µs) comandado pela ECU.
func (e *Engine) Step(tiUs float64) State {
	s := &e.st
	s.Tempo += e.dt
	s.RPM += (RPMAlvo(s.TPS) - s.RPM) * e.dt / TauRPM

	s.MassaArMg = MassaAr(s.TPS, s.RPM) + e.dist.VazamentoMg
	s.MassaCombMg = MassaCombustivel(tiUs, e.dist.FatorBico)
	s.LambdaCil = LambdaCilindro(s.MassaArMg, s.MassaCombMg)

	e.delay.Push(s.LambdaCil)
	s.Atraso = AtrasoTransporte(s.RPM)
	s.LambdaEscape = e.delay.At(s.Atraso)
	return *s
}

// MassaCombustivel é a massa injetada por ciclo, em mg, para um pulso tiUs.
func MassaCombustivel(tiUs, fatorBico float64) float64 {
	return fatorBico * QInjMgPerMs * max(0, tiUs-TMortoUs) / 1000
}

// LambdaCilindro é λ da mistura, limitado a [LambdaMin, LambdaMax].
func LambdaCilindro(arMg, combMg float64) float64 {
	if combMg <= 0 {
		return LambdaMax
	}
	return clamp(arMg/(combMg*AFRst), LambdaMin, LambdaMax)
}

// TiEstequiometrico é o pulso (µs) que dá λ = 1 com o bico nominal. Usado
// pelos testes e como referência da tabela da ECU.
func TiEstequiometrico(arMg float64) float64 {
	return TMortoUs + arMg/(AFRst*QInjMgPerMs)*1000
}
