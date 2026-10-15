// Grafico gera um SVG de uma corrida gravada pela planta (-csv): λ no
// cilindro e na sonda com a banda de ±0,02 em volta de 1, e o tempo de
// injeção. Linhas verticais marcam mudanças de perturbação.
//
//	grafico -o corrida.svg corrida.csv
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

type serie struct {
	nome, cor string
	y         []float64
}

type corrida struct {
	t                []float64
	lambdaCil, sonda []float64
	ti               []float64
	perturbacoes     []float64 // instantes em que bico ou vazamento mudaram
}

func main() {
	out := flag.String("o", "corrida.svg", "arquivo SVG de saída")
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("uso: grafico -o saida.svg corrida.csv")
	}
	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	c, err := ler(f)
	if err != nil {
		log.Fatal(err)
	}
	w, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()
	if err := desenhar(w, c); err != nil {
		log.Fatal(err)
	}
}

func ler(r io.Reader) (*corrida, error) {
	linhas, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(linhas) < 3 {
		return nil, fmt.Errorf("CSV sem dados")
	}
	col := map[string]int{}
	for i, nome := range linhas[0] {
		col[nome] = i
	}
	for _, nome := range []string{"tempo_s", "lambda_cilindro", "lambda_sonda", "ti_us", "fator_bico", "vazamento_mg"} {
		if _, ok := col[nome]; !ok {
			return nil, fmt.Errorf("CSV sem a coluna %s", nome)
		}
	}
	c := &corrida{}
	anterior := ""
	for _, l := range linhas[1:] {
		v := func(nome string) float64 {
			x, _ := strconv.ParseFloat(l[col[nome]], 64)
			return x
		}
		c.t = append(c.t, v("tempo_s"))
		c.lambdaCil = append(c.lambdaCil, v("lambda_cilindro"))
		c.sonda = append(c.sonda, v("lambda_sonda"))
		c.ti = append(c.ti, v("ti_us"))
		p := l[col["fator_bico"]] + "/" + l[col["vazamento_mg"]]
		if anterior != "" && p != anterior {
			c.perturbacoes = append(c.perturbacoes, v("tempo_s"))
		}
		anterior = p
	}
	return c, nil
}

const (
	largura   = 1000
	margemEsq = 70
	margemDir = 30
	alturaG   = 260 // altura de cada gráfico
	topo1     = 40
	topo2     = topo1 + alturaG + 60
	altura    = topo2 + alturaG + 50
)

func desenhar(w io.Writer, c *corrida) error {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" font-family="sans-serif" font-size="13">`+"\n", largura, altura)
	fmt.Fprintf(&b, `<rect width="100%%" height="100%%" fill="white"/>`+"\n")

	t0, t1 := c.t[0], c.t[len(c.t)-1]
	lmin, lmax := faixa(0.9, 1.1, c.lambdaCil, c.sonda)
	g1 := eixo{t0: t0, t1: t1, y0: lmin, y1: lmax, topo: topo1, titulo: "λ (alvo 1,00 ± 0,02)"}
	g1.desenhar(&b, c.perturbacoes)
	g1.faixaY(&b, 0.98, 1.02, "#d9f2d9")
	g1.linha(&b, c.t, c.lambdaCil, "#bbbbbb", "λ no cilindro")
	g1.linha(&b, c.t, c.sonda, "#1f5fbf", "λ na sonda")

	tmin, tmax := faixa(math.Inf(1), math.Inf(-1), c.ti)
	g2 := eixo{t0: t0, t1: t1, y0: tmin * 0.95, y1: tmax * 1.05, topo: topo2, titulo: "tempo de injeção (µs)"}
	g2.desenhar(&b, c.perturbacoes)
	g2.linha(&b, c.t, c.ti, "#c0392b", "")
	fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="middle">tempo (s)</text>`+"\n", largura/2, altura-10)
	b.WriteString("</svg>\n")
	_, err := io.WriteString(w, b.String())
	return err
}

func faixa(lo, hi float64, series ...[]float64) (float64, float64) {
	for _, s := range series {
		for _, v := range s {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	if hi <= lo {
		hi = lo + 1
	}
	return lo, hi
}

type eixo struct {
	t0, t1, y0, y1 float64
	topo           int
	titulo         string
	legendas       int
}

func (e *eixo) x(t float64) float64 {
	return margemEsq + (t-e.t0)/(e.t1-e.t0)*float64(largura-margemEsq-margemDir)
}

func (e *eixo) y(v float64) float64 {
	return float64(e.topo+alturaG) - (v-e.y0)/(e.y1-e.y0)*alturaG
}

func (e *eixo) desenhar(b *strings.Builder, marcas []float64) {
	fmt.Fprintf(b, `<text x="%d" y="%d" font-weight="bold">%s</text>`+"\n", margemEsq, e.topo-12, e.titulo)
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" fill="none" stroke="#333"/>`+"\n",
		margemEsq, e.topo, largura-margemEsq-margemDir, alturaG)
	for i := 0; i <= 5; i++ {
		v := e.y0 + (e.y1-e.y0)*float64(i)/5
		fmt.Fprintf(b, `<text x="%d" y="%.1f" text-anchor="end">%s</text>`+"\n", margemEsq-6, e.y(v)+4, rotulo(v))
		t := e.t0 + (e.t1-e.t0)*float64(i)/5
		fmt.Fprintf(b, `<text x="%.1f" y="%d" text-anchor="middle">%.1f</text>`+"\n", e.x(t), e.topo+alturaG+16, t)
	}
	for _, t := range marcas {
		fmt.Fprintf(b, `<line x1="%.1f" x2="%.1f" y1="%d" y2="%d" stroke="#e67e22" stroke-dasharray="4 3"/>`+"\n",
			e.x(t), e.x(t), e.topo, e.topo+alturaG)
	}
}

func (e *eixo) faixaY(b *strings.Builder, lo, hi float64, cor string) {
	fmt.Fprintf(b, `<rect x="%d" y="%.1f" width="%d" height="%.1f" fill="%s"/>`+"\n",
		margemEsq+1, e.y(hi), largura-margemEsq-margemDir-2, e.y(lo)-e.y(hi), cor)
}

// linha desenha a série, com no máximo ~1 ponto por pixel.
func (e *eixo) linha(b *strings.Builder, t, y []float64, cor, legenda string) {
	passo := max(1, len(t)/(largura-margemEsq-margemDir))
	fmt.Fprintf(b, `<polyline fill="none" stroke="%s" stroke-width="1.5" points="`, cor)
	for i := 0; i < len(t); i += passo {
		fmt.Fprintf(b, "%.1f,%.1f ", e.x(t[i]), e.y(y[i]))
	}
	b.WriteString("\"/>\n")
	if legenda != "" {
		lx := largura - margemDir - 160
		ly := e.topo + 18 + 18*e.legendas
		fmt.Fprintf(b, `<line x1="%d" x2="%d" y1="%d" y2="%d" stroke="%s" stroke-width="3"/>`, lx, lx+20, ly-4, ly-4, cor)
		fmt.Fprintf(b, `<text x="%d" y="%d">%s</text>`+"\n", lx+26, ly, legenda)
		e.legendas++
	}
}

func rotulo(v float64) string {
	if math.Abs(v) >= 100 {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}
