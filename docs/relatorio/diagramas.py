"""Diagramas do relatório (SVG escrito à mão, convertido para PNG com rsvg-convert)."""
import subprocess
from pathlib import Path

FONTE = "Liberation Sans, Arial, sans-serif"
CINZA = "#e8e8e8"


class SVG:
    def __init__(self, w, h):
        self.w, self.h = w, h
        self.el = []

    def caixa(self, x, y, w, h, texto, fill="white", tam=13, tracejada=False, negrito=False, rx=6):
        dash = ' stroke-dasharray="6 4"' if tracejada else ""
        self.el.append(f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="{rx}" fill="{fill}" '
                       f'stroke="#222" stroke-width="1.4"{dash}/>')
        self.texto(x + w / 2, y + h / 2, texto, tam, negrito)

    def losango(self, cx, cy, w, h, texto, tam=12):
        pts = f"{cx},{cy - h / 2} {cx + w / 2},{cy} {cx},{cy + h / 2} {cx - w / 2},{cy}"
        self.el.append(f'<polygon points="{pts}" fill="white" stroke="#222" stroke-width="1.4"/>')
        self.texto(cx, cy, texto, tam)

    def texto(self, x, y, texto, tam=13, negrito=False, ancora="middle"):
        linhas = texto.split("\n")
        peso = "bold" if negrito else "normal"
        y0 = y - (len(linhas) - 1) * tam * 0.6
        for i, l in enumerate(linhas):
            l = l.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
            self.el.append(f'<text x="{x}" y="{y0 + i * tam * 1.2}" font-family="{FONTE}" '
                           f'font-size="{tam}" font-weight="{peso}" text-anchor="{ancora}" '
                           f'dominant-baseline="middle">{l}</text>')

    def seta(self, *pts, duas=False, ponta=True, rotulo=None, pos_rotulo=None, tam=11):
        d = " ".join(f"{x},{y}" for x, y in pts)
        ini = ' marker-start="url(#ini)"' if duas else ""
        fim = ' marker-end="url(#fim)"' if ponta else ""
        self.el.append(f'<polyline points="{d}" fill="none" stroke="#222" stroke-width="1.4"'
                       f'{fim}{ini}/>')
        if rotulo:
            x, y = pos_rotulo
            self.texto(x, y, rotulo, tam)

    def salvar(self, caminho):
        defs = ('<defs>'
                '<marker id="fim" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto">'
                '<path d="M0,0 L10,4 L0,8 z" fill="#222"/></marker>'
                '<marker id="ini" markerWidth="10" markerHeight="8" refX="1" refY="4" orient="auto">'
                '<path d="M10,0 L0,4 L10,8 z" fill="#222"/></marker></defs>')
        svg = (f'<svg xmlns="http://www.w3.org/2000/svg" width="{self.w}" height="{self.h}" '
               f'viewBox="0 0 {self.w} {self.h}"><rect width="100%" height="100%" fill="white"/>'
               + defs + "".join(self.el) + "</svg>")
        caminho = Path(caminho)
        caminho.with_suffix(".svg").write_text(svg, encoding="utf-8")
        subprocess.run(["rsvg-convert", "-z", "2.5", "-o", str(caminho.with_suffix(".png")),
                        str(caminho.with_suffix(".svg"))], check=True)


def arquitetura(caminho):
    s = SVG(960, 560)
    # ECU no Renode
    s.caixa(20, 20, 400, 360, "", fill="#f6f6f6", tracejada=True)
    s.texto(220, 42, "Renode (Docker): STM32F072 emulado", 14, True)
    s.caixa(40, 70, 170, 130, "Firmware C + FreeRTOS\n\nControlTask (10 ms)\nCanTask (20 ms)")
    s.caixa(250, 70, 150, 50, "ADC + DMA\n(λ, TPS, RPM)", fill=CINZA)
    s.caixa(250, 150, 150, 50, "TIM2: pulso\ndo injetor", fill=CINZA)
    s.caixa(40, 260, 170, 50, "bxCAN 500 kbit/s", fill=CINZA)
    s.seta((250, 95), (210, 95))
    s.seta((210, 175), (250, 175))
    s.seta((125, 200), (125, 260), duas=True)
    # Planta
    s.caixa(540, 20, 400, 360, "", fill="#f6f6f6", tracejada=True)
    s.texto(740, 42, "Planta (Go): mestre do tempo", 14, True)
    s.caixa(560, 70, 170, 50, "Relógio da simulação\n(passo de 1 ms)")
    s.caixa(760, 70, 160, 50, "Motor e combustão\n(λ no cilindro)")
    s.caixa(760, 150, 160, 50, "Atraso do escape\n(buffer circular)")
    s.caixa(760, 230, 160, 50, "Sonda λ\n(1ª ordem, τ = 100 ms)")
    s.caixa(560, 230, 170, 50, "Sensores\n(goroutines)")
    s.caixa(560, 310, 170, 45, "Cliente CAN")
    s.seta((730, 95), (760, 95))
    s.seta((840, 120), (840, 150))
    s.seta((840, 200), (840, 230))
    s.seta((760, 255), (730, 255))
    # Lockstep entre os dois
    s.texto(480, 112, "monitor TCP", 11, True)
    s.seta((540, 140), (420, 140), rotulo="tensões → ADC", pos_rotulo=(480, 128))
    s.seta((420, 200), (540, 200), rotulo="largura do pulso", pos_rotulo=(480, 188))
    s.texto(480, 230, "lockstep\na cada 5 ms", 11)
    # Barramento CAN
    s.caixa(20, 420, 920, 36, "vcan0: barramento CAN virtual do Linux (SocketCAN)", fill=CINZA, tam=14)
    s.seta((125, 310), (125, 420), duas=True)
    s.seta((645, 355), (645, 420), duas=True)
    s.caixa(380, 494, 200, 50, "Painel (Go + raylib)")
    s.seta((480, 456), (480, 494), duas=True)
    s.salvar(caminho)


def fluxo_controle(caminho):
    s = SVG(500, 900)
    cx, w = 260, 300
    x = cx - w / 2

    def no(y, txt, fill="white", h=46):
        s.caixa(x, y, w, h, txt, fill=fill, tam=12)

    no(10, "TIM15 dispara o ADC (1 kHz)", CINZA)
    no(80, "ADC converte λ, TPS e RPM;\nDMA grava no buffer circular", CINZA)
    s.losango(cx, 200, 260, 70, "10 conjuntos\ncompletos?")
    no(270, "ISR do DMA:\nvTaskNotifyGiveFromISR", CINZA)
    no(340, "ControlTask desbloqueia\n(ulTaskNotifyTake)")
    no(410, "Média do bloco → λ, TPS, RPM\nt_base = tabela(TPS, RPM)")
    s.losango(cx, 530, 200, 70, "Malha\nfechada?")
    s.caixa(40, 590, 150, 46, "c = PI(λ − λ_alvo)", tam=12)
    s.caixa(330, 590, 150, 46, "c = 0", tam=12)
    no(670, "t_i = t_morto + t_base·(1 + c)\nlimitado a 90 % do ciclo")
    no(740, "Atualiza o TIM2\n(período e largura do pulso)")
    no(810, "Registra período, jitter\ne tempo de execução")
    for y0, y1 in [(56, 80), (126, 165), (235, 270), (316, 340), (386, 410), (456, 495)]:
        s.seta((cx, y0), (cx, y1))
    s.texto(cx + 14, 252, "sim", 11, ancora="start")
    s.seta((cx + 130, 200), (470, 200), (470, 33), (cx + w / 2, 33))
    s.texto(462, 188, "não", 11, ancora="end")
    s.seta((cx - 100, 530), (115, 530), (115, 590))
    s.texto(130, 518, "sim", 11)
    s.seta((cx + 100, 530), (405, 530), (405, 590))
    s.texto(390, 518, "não", 11)
    s.seta((115, 636), (115, 652), (cx, 652), (cx, 670))
    s.seta((405, 636), (405, 652), (cx, 652), ponta=False)
    s.seta((cx, 716), (cx, 740))
    s.seta((cx, 786), (cx, 810))
    s.seta((x, 833), (25, 833), (25, 363), (x, 363))
    s.texto(12, 600, "aguarda o próximo bloco", 11)
    s.el[-1] = s.el[-1].replace("<text ", '<text transform="rotate(-90 12 600)" ')
    s.h = 870
    s.salvar(caminho)


def fluxo_planta(caminho):
    s = SVG(420, 900)
    cx, w = 210, 300
    x = cx - w / 2

    def no(y, txt, fill="white", h=46):
        s.caixa(x, y, w, h, txt, fill=fill, tam=12)

    no(10, "Carrega ecu.elf no Renode\n(monitor TCP)", CINZA)
    no(80, "Aplica comandos do painel\n(CAN 0x310: acelerador, perturbações)")
    no(150, "Motor: rotação, ar admitido,\ncombustível e λ no cilindro")
    no(220, "Atraso do escape e sonda λ")
    no(290, "Sensores (goroutines) geram as\ntensões; barreira do passo")
    s.losango(cx, 410, 240, 70, "5 passos\nde 1 ms?")
    no(480, "lockstep: tensões → ADC,\nECU avança 5 ms", CINZA)
    no(550, "Lê a largura do pulso no TIM2\ne aplica ao bico da planta", CINZA)
    no(620, "Publica 0x300 (20 ms)\ne grava a amostra no CSV")
    for y0, y1 in [(56, 80), (126, 150), (196, 220), (266, 290), (336, 375), (445, 480), (526, 550)]:
        s.seta((cx, y0), (cx, y1))
    s.texto(cx + 14, 462, "sim", 11, ancora="start")
    s.seta((cx + 120, 410), (395, 410), (395, 173), (cx + w / 2, 173))
    s.texto(388, 398, "não", 11, ancora="end")
    s.seta((x, 643), (25, 643), (25, 103), (x, 103))
    s.texto(12, 380, "próximo passo de sincronização", 11)
    s.el[-1] = s.el[-1].replace("<text ", '<text transform="rotate(-90 12 380)" ')
    s.h = 680
    s.salvar(caminho)


def gerar(pasta):
    pasta = Path(pasta)
    pasta.mkdir(exist_ok=True)
    arquitetura(pasta / "arquitetura")
    fluxo_controle(pasta / "fluxo-controle")
    fluxo_planta(pasta / "fluxo-planta")


if __name__ == "__main__":
    gerar(Path(__file__).parent / "fig")
