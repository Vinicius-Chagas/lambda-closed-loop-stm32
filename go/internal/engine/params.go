// Package engine é o modelo físico da planta: motor, combustão e escape
// (docs/equacoes.md, seções 2 e 3). A sonda e os demais sensores ficam em
// internal/sensors.
package engine

// Parâmetros do modelo. Mantenha em sincronia com firmware/control/params.h.
const (
	AFRst = 14.7 // relação ar/combustível estequiométrica

	MArMaxMg = 480.0 // massa de ar máxima por cilindro, por ciclo (ρ·V_cil)
	RPMLenta = 800.0
	RPMMax   = 6000.0
	TauRPM   = 0.5 // s

	QInjMgPerMs = 3.0 // vazão do bico
	TMortoUs    = 800.0

	// Atraso de transporte: Td = AtrasoCiclo/RPM + AtrasoFixo.
	AtrasoCiclo = 180.0 // s·rpm
	AtrasoFixo  = 0.020 // s

	// λ no cilindro fica limitado a esta faixa (sem combustível → λ_max).
	LambdaMin = 0.5
	LambdaMax = 2.0
)

// FracaoBorboleta é f(TPS): fração do ar máximo que passa pela borboleta.
func FracaoBorboleta(tps float64) float64 {
	return 0.10 + 0.90*clamp(tps, 0, 100)/100
}

// Eficiencia é η(RPM), a eficiência volumétrica.
func Eficiencia(rpm float64) float64 {
	x := (rpm - 4000) / 3000
	return 0.85 - 0.10*x*x
}

// MassaAr é a massa de ar nominal por cilindro e ciclo, em mg (sem vazamento).
func MassaAr(tps, rpm float64) float64 {
	return MArMaxMg * FracaoBorboleta(tps) * Eficiencia(rpm)
}

// RPMAlvo é a rotação de equilíbrio para a abertura da borboleta.
func RPMAlvo(tps float64) float64 {
	return RPMLenta + (RPMMax-RPMLenta)*clamp(tps, 0, 100)/100
}

// AtrasoTransporte é o tempo, em s, entre a combustão e a chegada do gás à sonda.
func AtrasoTransporte(rpm float64) float64 {
	return AtrasoCiclo/max(rpm, 300) + AtrasoFixo
}

func clamp(x, lo, hi float64) float64 {
	return min(max(x, lo), hi)
}
