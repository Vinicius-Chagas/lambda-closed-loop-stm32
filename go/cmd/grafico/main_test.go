package main

import (
	"strings"
	"testing"
)

const csvTeste = `tempo_s,tps_pct,rpm,lambda_cilindro,lambda_escape,lambda_sonda,ti_us,ciclo_us,fator_bico,vazamento_mg,atraso_ms
0.000,23.0,2000,1.0000,1.0000,1.0000,2900,60000,1.000,0.0,110.0
0.005,23.0,2000,1.2500,1.0000,1.0000,2900,60000,0.800,0.0,110.0
0.010,23.0,2000,1.1000,1.2000,1.0500,3200,60000,0.800,0.0,110.0
`

func TestGrafico(t *testing.T) {
	c, err := ler(strings.NewReader(csvTeste))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.t) != 3 || c.sonda[2] != 1.05 || c.ti[2] != 3200 {
		t.Fatalf("leitura: %+v", c)
	}
	if len(c.perturbacoes) != 1 || c.perturbacoes[0] != 0.005 {
		t.Errorf("perturbações %v, quer [0.005]", c.perturbacoes)
	}
	var b strings.Builder
	if err := desenhar(&b, c); err != nil {
		t.Fatal(err)
	}
	svg := b.String()
	if n := strings.Count(svg, "<polyline"); n != 3 {
		t.Errorf("%d curvas, quer 3", n)
	}
	for _, s := range []string{"λ na sonda", "tempo de injeção", "stroke-dasharray", "</svg>"} {
		if !strings.Contains(svg, s) {
			t.Errorf("SVG sem %q", s)
		}
	}
}

func TestGraficoCSVInvalido(t *testing.T) {
	if _, err := ler(strings.NewReader("a,b\n1,2\n3,4\n")); err == nil {
		t.Error("aceitou CSV sem as colunas")
	}
}
