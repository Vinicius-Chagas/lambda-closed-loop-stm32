// Painel da bancada: uma tela com a mistura (λ), os números principais e o
// gráfico dos últimos 10 s. Lê e comanda a bancada pelo CAN (vcan0).
//
//	painel                    # bancada real no vcan0
//	painel -demo              # sem Renode nem vcan0: controle C (SIL) dentro do painel
//	painel -tela-cheia
//
// Teclas: ↑/↓ acelerador, 1 bico −20 %, 2 vazamento de ar, espaço malha
// aberta/fechada, F tela cheia, Esc sai.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/vinicius-chagas/lambda-closed-loop-stm32/go/internal/painel"
)

//go:embed fonte/DejaVuSans-Bold.ttf
var fonteTTF []byte

func init() { runtime.LockOSThread() } // raylib só pode ser chamada da thread principal

func main() {
	iface := flag.String("can", "vcan0", "interface SocketCAN da bancada")
	demo := flag.Bool("demo", false, "simula a bancada dentro do painel (sem Renode e sem vcan0)")
	telaCheia := flag.Bool("tela-cheia", false, "abre em tela cheia")
	captura := flag.String("captura", "", "salva a tela neste PNG e sai (aplica o bico −20 % na metade do tempo)")
	apos := flag.Float64("apos", 8, "segundos até a captura")
	flag.Parse()

	var src fonte
	if *demo {
		src = abrirDemo()
	} else {
		c, err := abrirCAN(*iface)
		if err != nil {
			log.Fatalf("abrindo %s: %v (rode scripts/setup-vcan.sh, ou use -demo)", *iface, err)
		}
		src = c
	}
	defer src.Close()

	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint | rl.FlagVsyncHint)
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.InitWindow(baseL, baseA, "Bancada λ — painel")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)
	if *telaCheia {
		rl.ToggleFullscreen()
	}
	t := novaTela()
	defer t.fechar()

	var e painel.Estado
	inicio := time.Now()
	perturbou := false
	for !rl.WindowShouldClose() {
		agora := time.Now()
		for lido := true; lido; {
			select {
			case f, ok := <-src.Frames():
				if ok {
					e.Atualizar(f, agora)
				}
				lido = ok
			default:
				lido = false
			}
		}
		for _, k := range teclas() {
			if err := src.Enviar(e.Comando(k)); err != nil {
				log.Printf("enviando comando: %v", err)
			}
		}
		if rl.IsKeyPressed(rl.KeyF) {
			rl.ToggleFullscreen()
		}

		decorrido := agora.Sub(inicio).Seconds()
		if *captura != "" && !perturbou && decorrido >= *apos/2 {
			src.Enviar(e.Comando(painel.AlternarBico))
			perturbou = true
		}

		rl.BeginDrawing()
		t.desenhar(&e, agora)
		rl.EndDrawing()

		if *captura != "" && decorrido >= *apos {
			img := rl.LoadImageFromScreen()
			ok := rl.ExportImage(*img, *captura)
			rl.UnloadImage(img)
			if !ok {
				log.Fatalf("não consegui salvar %s", *captura)
			}
			log.Printf("tela salva em %s", *captura)
			return
		}
	}
}

func teclas() []painel.Tecla {
	var ks []painel.Tecla
	mapa := []struct {
		k int32
		t painel.Tecla
	}{
		{rl.KeyUp, painel.AceleradorMais}, {rl.KeyDown, painel.AceleradorMenos},
		{rl.KeyOne, painel.AlternarBico}, {rl.KeyKp1, painel.AlternarBico},
		{rl.KeyTwo, painel.AlternarVazamento}, {rl.KeyKp2, painel.AlternarVazamento},
		{rl.KeySpace, painel.AlternarMalha},
	}
	for _, m := range mapa {
		if rl.IsKeyPressed(m.k) {
			ks = append(ks, m.t)
		}
	}
	return ks
}

// número com vírgula decimal
func num(v float64, casas int) string {
	return strings.Replace(fmt.Sprintf("%.*f", casas, v), ".", ",", 1)
}
