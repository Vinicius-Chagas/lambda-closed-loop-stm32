package main

import (
	"fmt"
	"image/color"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/canbus"
	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/painel"
)

// Layout pensado em 1280×720 e escalado para a janela.
const (
	baseL = 1280
	baseA = 720
)

var (
	corFundo  = rl.NewColor(0x10, 0x14, 0x18, 255)
	corCartao = rl.NewColor(0x1c, 0x23, 0x2b, 255)
	corTexto  = rl.NewColor(0xec, 0xef, 0xf1, 255)
	corFraco  = rl.NewColor(0x90, 0xa4, 0xae, 255)
	corRica   = rl.NewColor(0xff, 0x70, 0x43, 255)
	corIdeal  = rl.NewColor(0x66, 0xbb, 0x6a, 255)
	corPobre  = rl.NewColor(0x42, 0xa5, 0xf5, 255)
	corSonda  = rl.NewColor(0xff, 0xca, 0x28, 255)
	corCil    = rl.NewColor(0x78, 0x90, 0x9c, 255)
	corAlerta = rl.NewColor(0xff, 0xa7, 0x26, 255)
)

func corMistura(m painel.Mistura) color.RGBA {
	return [...]color.RGBA{corRica, corIdeal, corPobre}[m]
}

type tela struct {
	pequena, grande rl.Font
	s, ox, oy       float32 // escala e deslocamento do layout base
}

func novaTela() *tela {
	var cps []rune
	for r := rune(32); r < 127; r++ {
		cps = append(cps, r)
	}
	for r := rune(160); r < 256; r++ {
		cps = append(cps, r)
	}
	cps = append(cps, 'λ', '−', '↑', '↓', '·', '⚠')
	t := &tela{
		pequena: rl.LoadFontFromMemory(".ttf", fonteTTF, 48, cps),
		grande:  rl.LoadFontFromMemory(".ttf", fonteTTF, 160, cps),
	}
	rl.SetTextureFilter(t.pequena.Texture, rl.FilterBilinear)
	rl.SetTextureFilter(t.grande.Texture, rl.FilterBilinear)
	return t
}

func (t *tela) fechar() {
	rl.UnloadFont(t.pequena)
	rl.UnloadFont(t.grande)
}

// coordenadas do layout base → tela
func (t *tela) p(x, y float32) rl.Vector2 { return rl.Vector2{X: t.ox + x*t.s, Y: t.oy + y*t.s} }
func (t *tela) r(x, y, w, h float32) rl.Rectangle {
	return rl.Rectangle{X: t.ox + x*t.s, Y: t.oy + y*t.s, Width: w * t.s, Height: h * t.s}
}

type alinhamento int

const (
	esq alinhamento = iota
	centro
	dir
)

func (t *tela) texto(s string, x, y, tam float32, c color.RGBA, al alinhamento) {
	f := t.pequena
	if tam > 60 {
		f = t.grande
	}
	larg := rl.MeasureTextEx(f, s, tam*t.s, 0).X / t.s
	switch al {
	case centro:
		x -= larg / 2
	case dir:
		x -= larg
	}
	rl.DrawTextEx(f, s, t.p(x, y), tam*t.s, 0, c)
}

func (t *tela) cartao(x, y, w, h float32) {
	rl.DrawRectangleRounded(t.r(x, y, w, h), 0.06, 8, corCartao)
}

func (t *tela) desenhar(e *painel.Estado, agora time.Time) {
	W, H := float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())
	t.s = min(W/baseL, H/baseA)
	t.ox, t.oy = (W-baseL*t.s)/2, (H-baseA*t.s)/2
	rl.ClearBackground(corFundo)

	t.cabecalho(e, agora)
	t.lambda(e)
	t.numeros(e)
	t.grafico(e)
	t.rodape()
}

func (t *tela) cabecalho(e *painel.Estado, agora time.Time) {
	t.texto("Mistura ar-combustível · controle em malha fechada", 24, 18, 30, corTexto, esq)
	modo, cor := "MALHA FECHADA", corIdeal
	if e.Estado.Modo == canbus.MalhaAberta {
		modo, cor = "MALHA ABERTA", corAlerta
	}
	if e.NEstado == 0 {
		modo, cor = "—", corFraco
	}
	rl.DrawRectangleRounded(t.r(1016, 12, 240, 44), 0.3, 8, cor)
	t.texto(modo, 1136, 21, 26, corFundo, centro)
	if !e.Conectado(agora) {
		t.texto("SEM SINAL DA BANCADA", 1000, 22, 24, corRica, dir)
	}
}

func (t *tela) lambda(e *painel.Estado) {
	t.cartao(24, 72, 600, 270)
	alvo := e.Estado.LambdaAlvo
	if alvo == 0 {
		alvo = 1
	}
	l := e.Planta.LambdaSonda
	m := painel.Classificar(l, alvo)
	t.texto("λ na sonda", 44, 86, 24, corFraco, esq)
	t.texto("alvo "+num(alvo, 2), 604, 86, 24, corFraco, dir)
	if e.NPlanta == 0 {
		t.texto("—", 44, 104, 130, corFraco, esq)
	} else {
		t.texto(num(l, 3), 44, 104, 130, corMistura(m), esq)
		t.texto(m.String(), 604, 150, 48, corMistura(m), dir)
	}

	// barra rica | ideal | pobre
	x0, x1, y, h := float32(44), float32(604), float32(258), float32(34)
	px := func(v float64) float32 { return x0 + float32(painel.PosicaoBarra(v))*(x1-x0) }
	ri, pi := px(alvo-painel.BandaIdeal), px(alvo+painel.BandaIdeal)
	rl.DrawRectangleRec(t.r(x0, y, ri-x0, h), corRica)
	rl.DrawRectangleRec(t.r(ri, y, pi-ri, h), corIdeal)
	rl.DrawRectangleRec(t.r(pi, y, x1-pi, h), corPobre)
	if e.NPlanta > 0 {
		mx := px(l)
		rl.DrawTriangle(t.p(mx, y-4), t.p(mx+12, y-22), t.p(mx-12, y-22), corTexto)
		rl.DrawRectangleRec(t.r(mx-3, y-4, 6, h+8), corTexto)
	}
	t.texto("RICA", x0, y+h+8, 22, corRica, esq)
	t.texto("IDEAL", (x0+x1)/2, y+h+8, 22, corIdeal, centro)
	t.texto("POBRE", x1, y+h+8, 22, corPobre, dir)
}

func (t *tela) numeros(e *painel.Estado) {
	t.cartao(644, 72, 612, 270)
	type valor struct {
		rotulo, texto string
		cor           color.RGBA
	}
	vazio := e.NEstado == 0
	val := func(s string) string {
		if vazio {
			return "—"
		}
		return s
	}
	bico, corBico := fmt.Sprintf("%d %%", e.Planta.VazaoBicoPct), corTexto
	if e.BicoPerturbado() {
		corBico = corAlerta
	}
	vaz, corVaz := fmt.Sprintf("%d mg", e.Planta.VazamentoMg), corTexto
	if e.VazamentoAtivo() {
		corVaz = corAlerta
	}
	if e.NPlanta == 0 {
		bico, vaz = "—", "—"
	}
	vs := []valor{
		{"Rotação", val(fmt.Sprintf("%.0f rpm", e.Sensores.RPM)), corTexto},
		{"Acelerador", val(fmt.Sprintf("%.0f %%", e.Sensores.TPS)), corTexto},
		{"Tempo de injeção", val(fmt.Sprintf("%d µs", e.Estado.TiUs)), corTexto},
		{"Correção do controle", val(sinal(e.Estado.Correcao*100) + " %"), corTexto},
		{"Vazão do bico", bico, corBico},
		{"Vazamento de ar", vaz, corVaz},
	}
	for i, v := range vs {
		x := float32(664 + (i%2)*300)
		y := float32(84 + (i/2)*84)
		t.texto(v.rotulo, x, y, 20, corFraco, esq)
		t.texto(v.texto, x, y+24, 40, v.cor, esq)
	}

	tm := "ControlTask: aguardando 0x110"
	if e.NTempos > 0 {
		tm = fmt.Sprintf("ControlTask: período %s ms · jitter %d µs · execução %d µs · deadlines perdidos %d",
			num(float64(e.Tempos.PeriodoUs)/1000, 1), e.Tempos.JitterMaxUs, e.Tempos.ExecMaxUs, e.Tempos.DeadlinesPerdidos)
	} else if e.NEstado > 0 {
		tm = "ControlTask: tempos só com a ECU no Renode"
	}
	t.texto(tm, 24, 352, 20, corFraco, esq)
	if e.NPlanta > 0 {
		t.texto(fmt.Sprintf("atraso do escape até a sonda: %d ms", e.Planta.AtrasoMs), 1256, 352, 20, corFraco, dir)
	}
}

func sinal(v float64) string {
	if v > 0.05 {
		return "+" + num(v, 1)
	}
	if v < -0.05 {
		return "−" + num(-v, 1)
	}
	return num(0, 1)
}

const (
	gx, gy, gw, gh = float32(24), float32(384), float32(1232), float32(272)
	lMin, lMax     = 0.70, 1.30
)

func (t *tela) grafico(e *painel.Estado) {
	t.cartao(gx, gy, gw, gh)
	ix, iy, iw, ih := gx+70, gy+40, gw-90, gh-56 // área das curvas
	ypos := func(l float64) float32 {
		l = min(max(l, lMin), lMax)
		return iy + ih - float32((l-lMin)/(lMax-lMin))*ih
	}
	alvo := e.Estado.LambdaAlvo
	if alvo == 0 {
		alvo = 1
	}
	y0, y1 := ypos(alvo+painel.BandaIdeal), ypos(alvo-painel.BandaIdeal)
	rl.DrawRectangleRec(t.r(ix, y0, iw, y1-y0), rl.ColorAlpha(corIdeal, 0.25))
	for _, l := range []float64{0.7, 0.8, 0.9, 1.0, 1.1, 1.2, 1.3} {
		rl.DrawLineEx(t.p(ix, ypos(l)), t.p(ix+iw, ypos(l)), 1, rl.ColorAlpha(corFraco, 0.35))
		t.texto(num(l, 2), ix-10, ypos(l)-10, 18, corFraco, dir)
	}
	t.texto("λ nos últimos 10 s", ix, gy+10, 22, corTexto, esq)
	legendas := []struct {
		nome string
		cor  color.RGBA
	}{{"λ no cilindro", corCil}, {"λ na sonda", corSonda}, {"alvo", corIdeal}}
	lx := ix + iw
	for i := len(legendas) - 1; i >= 0; i-- {
		lg := legendas[i]
		larg := rl.MeasureTextEx(t.pequena, lg.nome, 20*t.s, 0).X / t.s
		lx -= larg
		t.texto(lg.nome, lx, gy+12, 20, corFraco, esq)
		lx -= 30
		rl.DrawRectangleRec(t.r(lx, gy+20, 22, 5), lg.cor)
		lx -= 24
	}

	h := e.Historico()
	if len(h) < 2 {
		return
	}
	fim := h[len(h)-1].T
	xpos := func(tt float64) float32 { return ix + iw - float32((fim-tt)/painel.Janela)*iw }
	cil := make([]rl.Vector2, len(h))
	sonda := make([]rl.Vector2, len(h))
	alvos := make([]rl.Vector2, len(h))
	for i, a := range h {
		cil[i] = t.p(xpos(a.T), ypos(a.LambdaCilindro))
		sonda[i] = t.p(xpos(a.T), ypos(a.LambdaSonda))
		al := a.LambdaAlvo
		if al == 0 {
			al = 1
		}
		alvos[i] = t.p(xpos(a.T), ypos(al))
	}
	rl.DrawSplineLinear(alvos, 2*t.s, corIdeal)
	rl.DrawSplineLinear(cil, 2*t.s, corCil)
	rl.DrawSplineLinear(sonda, 4*t.s, corSonda)
	for s := 0; s <= int(painel.Janela); s += 2 {
		x := ix + iw - float32(s)/float32(painel.Janela)*iw
		rotulo := fmt.Sprintf("−%d s", s)
		if s == 0 {
			rotulo = "agora"
		}
		t.texto(rotulo, x, iy+ih+2, 16, corFraco, centro)
	}
}

func (t *tela) rodape() {
	itens := []struct{ tecla, acao string }{
		{"↑ ↓", "acelerador"}, {"1", "bico −20 %"}, {"2", "vazamento de ar"},
		{"ESPAÇO", "malha aberta/fechada"}, {"F", "tela cheia"}, {"ESC", "sair"},
	}
	x := float32(24)
	for _, it := range itens {
		lt := rl.MeasureTextEx(t.pequena, it.tecla, 20*t.s, 0).X/t.s + 18
		rl.DrawRectangleRounded(t.r(x, 672, lt, 32), 0.3, 6, corCartao)
		t.texto(it.tecla, x+lt/2, 677, 20, corTexto, centro)
		x += lt + 8
		t.texto(it.acao, x, 677, 20, corFraco, esq)
		x += rl.MeasureTextEx(t.pequena, it.acao, 20*t.s, 0).X/t.s + 26
	}
}
