package engine

import (
	"math"
	"testing"
)

func near(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %.5f, esperado %.5f ± %g", name, got, want, tol)
	}
}

func TestCurvas(t *testing.T) {
	near(t, "η(4000)", Eficiencia(4000), 0.85, 1e-9)
	near(t, "η(800)", Eficiencia(800), 0.85-0.10*(3200.0/3000)*(3200.0/3000), 1e-9)
	near(t, "f(0)", FracaoBorboleta(0), 0.10, 1e-9)
	near(t, "f(100)", FracaoBorboleta(100), 1.0, 1e-9)
	near(t, "f(150) satura", FracaoBorboleta(150), 1.0, 1e-9)
	near(t, "RPMAlvo(0)", RPMAlvo(0), 800, 1e-9)
	near(t, "RPMAlvo(50)", RPMAlvo(50), 3400, 1e-9)
}

func TestAtrasoDeTransporte(t *testing.T) {
	// Tabela de docs/equacoes.md.
	near(t, "Td(800)", AtrasoTransporte(800), 0.245, 1e-9)
	near(t, "Td(2000)", AtrasoTransporte(2000), 0.110, 1e-9)
	near(t, "Td(6000)", AtrasoTransporte(6000), 0.050, 1e-9)
	near(t, "Td(0) limitado", AtrasoTransporte(0), 180.0/300+0.02, 1e-9)
}

func TestLambdaEstequiometrico(t *testing.T) {
	ar := MassaAr(20, 2000)
	ti := TiEstequiometrico(ar)
	near(t, "λ", LambdaCilindro(ar, MassaCombustivel(ti, 1)), 1.0, 1e-9)
	// Bico entregando 20 % a menos: λ = 1/0,8.
	near(t, "λ bico sujo", LambdaCilindro(ar, MassaCombustivel(ti, 0.8)), 1.25, 1e-9)
	// Abaixo do tempo morto não sai combustível.
	near(t, "λ sem combustível", LambdaCilindro(ar, MassaCombustivel(TMortoUs, 1)), LambdaMax, 0)
}

func TestDelayLine(t *testing.T) {
	d := NewDelayLine(0.001, 0.5, 0)
	for i := 1; i <= 300; i++ {
		d.Push(float64(i)) // amostra i no instante i ms
	}
	near(t, "agora", d.At(0), 300, 1e-9)
	near(t, "100 ms atrás", d.At(0.100), 200, 1e-9)
	near(t, "interpolado", d.At(0.1005), 199.5, 1e-9)
	// Mais antigo que o histórico: devolve o mais antigo guardado.
	near(t, "fora do buffer", d.At(10), d.At(0.5), 1e-9)
}

func TestRotacaoSegueBorboleta(t *testing.T) {
	e := New(0.001)
	e.SetTPS(50)
	var st State
	for range int(TauRPM / 0.001) { // uma constante de tempo
		st = e.Step(0)
	}
	// 1ª ordem: 63,2 % do degrau após τ.
	near(t, "RPM após τ", st.RPM, 800+0.632*(3400-800), 5)
}

func TestAtrasoNoEscape(t *testing.T) {
	const dt = 0.001
	e := New(dt)
	ti := TiEstequiometrico(MassaAr(0, RPMLenta))
	for range 1000 {
		e.Step(ti)
	}
	near(t, "λ escape em regime", e.State().LambdaEscape, 1, 1e-3)

	// Degrau no injetor: λ do cilindro muda na hora, o do escape só depois de Td.
	e.SetDisturbances(Disturbances{FatorBico: 0.8})
	td := AtrasoTransporte(RPMLenta)
	var mudou float64 = -1
	for i := 1; i <= 500; i++ {
		st := e.Step(ti)
		if i == 1 {
			near(t, "λ cilindro logo após o degrau", st.LambdaCil, 1.25, 1e-3)
		}
		if mudou < 0 && st.LambdaEscape > 1.1 {
			mudou = float64(i) * dt
		}
	}
	near(t, "atraso observado", mudou, td, 0.003)
}

func TestVazamentoEmpobrece(t *testing.T) {
	e := New(0.001)
	e.SetTPS(0)
	ti := TiEstequiometrico(MassaAr(0, RPMLenta))
	e.SetDisturbances(Disturbances{FatorBico: 1, VazamentoMg: 10})
	st := e.Step(ti)
	if st.LambdaCil <= 1.1 {
		t.Fatalf("vazamento de 10 mg em marcha lenta deveria empobrecer, λ = %.3f", st.LambdaCil)
	}
}
