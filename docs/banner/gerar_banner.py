"""Preenche o modelo de banner da Mostra de Projetos (docs/banner/modelo.pptx).

    python3 docs/banner/gerar_banner.py

Gera banner.pptx e banner.pdf (LibreOffice). Requer python-pptx e soffice.
"""
import copy
import re
import subprocess
from pathlib import Path

from lxml import etree
from pptx import Presentation
from pptx.dml.color import RGBColor
from pptx.enum.text import MSO_ANCHOR, PP_ALIGN
from pptx.util import Cm, Pt

AQUI = Path(__file__).resolve().parent
RAIZ = AQUI.parent.parent
FIG = RAIZ / "docs" / "relatorio" / "fig"

CORPO = 24        # tamanho único do corpo (o modelo pede 20 a 32 pt)
FONTE_NOTA = 20   # "Fonte: Elaborado pelos autores." (como no modelo)
PRETO = RGBColor(0, 0, 0)
CINZA = RGBColor(0xE7, 0xE6, 0xE6)
ESQ, DIR, LARG = 1.0, 30.5, 27.9  # colunas (cm)

IDENT = "CP120TIN2 - Grupo XX"
TITULO = "MODELAMENTO DO CONTROLE EM MALHA FECHADA DA MISTURA AR-COMBUSTÍVEL AUTOMOTIVA"
NOMES = [
    ["Eduardo de Souza Monteiro – 212123",
     "Fellipe Alessandro Scruph – 222378",
     "Guilherme Matheus Fonseca Waizbart – 211899"],
    ["João Victor de Oliveira – 212106",
     "Renan Vizoto Ferreira – 222220",
     "Vinicius Cândido das Chagas – 211873"],
]
ORIENTADORES = "Orientadores: Prof. Marcos Fábio Jardini e Prof. Adilson Rocha"

INTRODUCAO = (
    "O fator lambda (λ) mede a proporção entre ar e combustível na queima. Com λ = 1 a mistura é "
    "a ideal e o catalisador de três vias opera com máxima eficiência; com λ < 1 a mistura é rica "
    "(mais consumo e emissões de CO e HC) e com λ > 1, pobre (mais NO_{x} e falhas de combustão). "
    "A central eletrônica (ECU) mantém λ = 1 em malha fechada, lendo a sonda lambda no escapamento "
    "e corrigindo o tempo de injeção. O desafio é o atraso: o gás leva de 50 a 250 ms para chegar à "
    "sonda, conforme a rotação, o que pode desestabilizar o controle. Este projeto constrói uma "
    "bancada simulada, sem motor real: o firmware em C com FreeRTOS de um STM32F072 roda no "
    "emulador Renode, e uma planta em Go simula motor, escapamento e sonda (Figura 1)."
)
EQUACOES = [
    "λ = m_{ar} / (14,7 · m_{comb})",
    "T_{d} = 180 / RPM + 0,02 s     sonda de 1ª ordem, τ = 100 ms",
    "t_{i} = t_{morto} + t_{base}(TPS, RPM) · (1 + c)",
    "c = 0,15 · e + 2,0 · ∫e dt,   e = λ_{med} − λ_{alvo},   |c| ≤ 30 %",
]
MODELO_INTRO = ("A planta calcula o λ no cilindro a cada 1 ms, aplica o atraso do escapamento "
                "e a dinâmica da sonda. A ECU soma uma tabela de alimentação a uma correção PI:")
TAREFAS = (
    "O TIM15 dispara o ADC a cada 1 ms e o DMA grava λ, TPS e RPM sem usar a CPU. A cada 10 "
    "amostras, o DMA notifica a ControlTask (prioridade 3), que calcula t_{i} e atualiza o TIM2 "
    "(pulso do injetor, resolução de 1 µs). A CanTask (prioridade 2, 20 ms) publica o estado no "
    "CAN e recebe comandos por uma fila. Metas da ControlTask: jitter ≤ 100 µs, execução ≤ 1 ms e "
    "nenhum prazo perdido."
)
RESULTADOS = (
    "A bancada foi validada em três níveis: testes unitários (Unity e Go), integração (o controle C "
    "dentro da planta e o firmware completo no Renode) e ponta a ponta pelo barramento CAN virtual. "
    "As métricas são o período, o jitter e o tempo de execução da tarefa de controle, os prazos "
    "perdidos e, na malha, o tempo para λ voltar a 1,00 ± 0,02 e o sobressinal após reduzir em 20 % "
    "a vazão do bico a 2000 rpm (Figura 2)."
)
CONCLUSAO = (
    "A bancada atingiu todas as metas: a tarefa de controle usou menos de 2 % da CPU, com jitter dez "
    "vezes menor que o limite, e a mistura voltou a λ = 1 em 1,3 s, sem sobressinal. Amostrar por "
    "hardware, acordar a tarefa por notificação e deixar a comunicação em prioridade menor manteve o "
    "caminho crítico curto e previsível. O próximo passo é gravar o mesmo binário em uma placa "
    "STM32 real e repetir as medições."
)
CUSTO = [  # etapa, horas (1 programador a R$ 50/h)
    ("Modelagem", 25), ("Firmware", 30), ("Planta e painel", 35), ("Testes e docs", 30),
]
METAS = [
    ("Período", "10 ms", "10 ms"),
    ("Jitter", "≤ 100 µs", "≤ 10 µs"),
    ("Execução", "≤ 1 ms", "0,16 ms"),
    ("Prazos perdidos", "0", "0"),
    ("Acomodação", "≤ 2 s", "1,3 s"),
    ("Sobressinal", "≤ 5 %", "0 %"),
]

# Caixas de instrução do modelo que dão lugar ao conteúdo.
REMOVER = {"CaixaDeTexto 2", "CaixaDeTexto 7", "CaixaDeTexto 11", "CaixaDeTexto 23",
           "CaixaDeTexto 37", "CaixaDeTexto 38", "CaixaDeTexto 39", "CaixaDeTexto 41",
           "CaixaDeTexto 42", "CaixaDeTexto 59", "CaixaDeTexto 60", "CaixaDeTexto 61",
           "CaixaDeTexto 4", "CaixaDeTexto 5", "CaixaDeTexto 6"}


def reais(v):
    return f"R$ {v:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")


def runs_ricos(par, s, tam, negrito=False):
    """Texto com _{subscrito}."""
    for tok in re.split(r"(_\{[^}]*\})", s):
        if not tok:
            continue
        r = par.add_run()
        r.text = tok[2:-1] if tok.startswith("_{") else tok
        r.font.size = Pt(tam)
        r.font.name = "Arial"
        r.font.bold = negrito
        r.font.color.rgb = PRETO
        if tok.startswith("_{"):
            r.font._rPr.set("baseline", "-25000")


def trocar_texto(shape, linhas, tam=None):
    """Mantém a formatação do 1º run da caixa do modelo e troca o texto."""
    tf = shape.text_frame
    modelo = copy.deepcopy(tf.paragraphs[0]._p)
    for p in list(tf.paragraphs):
        p._p.getparent().remove(p._p)
    for linha in linhas:
        p = copy.deepcopy(modelo)
        for r in p.findall("{http://schemas.openxmlformats.org/drawingml/2006/main}r")[1:]:
            p.remove(r)
        tf._txBody.append(p)
        par = tf.paragraphs[-1]
        par.runs[0].text = linha
        if tam:
            par.runs[0].font.size = Pt(tam)


class Banner:
    def __init__(self):
        self.prs = Presentation(AQUI / "modelo.pptx")
        self.sl = self.prs.slides[0]
        self.por_nome = {sh.name: sh for sh in self.sl.shapes}

    def texto(self, x, y, w, h, paragrafos, alinhamento=PP_ALIGN.JUSTIFY, tam=CORPO,
              negrito=False, espaco=0):
        tb = self.sl.shapes.add_textbox(Cm(x), Cm(y), Cm(w), Cm(h))
        tf = tb.text_frame
        tf.word_wrap = True
        tf.margin_left = tf.margin_right = Cm(0.1)
        tf.margin_top = tf.margin_bottom = Cm(0.05)
        tf.vertical_anchor = MSO_ANCHOR.TOP
        for i, s in enumerate(paragrafos):
            par = tf.paragraphs[0] if i == 0 else tf.add_paragraph()
            par.alignment = alinhamento
            par.line_spacing = 1.0
            par.space_after = Pt(espaco)
            runs_ricos(par, s, tam, negrito)
        return tb

    def legenda(self, x, y, w, rotulo, txt):
        tb = self.texto(x, y, w, 1.2, [rotulo + ". "], PP_ALIGN.CENTER, negrito=True)
        runs_ricos(tb.text_frame.paragraphs[0], txt, CORPO)

    def fonte(self, x, y, w):
        self.texto(x, y, w, 0.9, ["Fonte: Elaborado pelos autores."], PP_ALIGN.CENTER, FONTE_NOTA)

    def figura(self, x, y, w, rotulo, txt, caminho, largura):
        """Legenda, imagem centralizada na largura w e fonte. Retorna o y final."""
        self.legenda(x, y, w, rotulo, txt)
        pic = self.sl.shapes.add_picture(str(caminho), Cm(x + (w - largura) / 2), Cm(y + 1.3),
                                         width=Cm(largura))
        fim = y + 1.3 + pic.height / 360000
        self.fonte(x, fim + 0.1, w)
        return fim + 1.1

    def tabela(self, x, y, w, rotulo, txt, cab, linhas, larguras, negrito_ultima=False,
               alt_linha=1.2):
        self.legenda(x, y, w, rotulo, txt)
        larg_tab = sum(larguras)
        x0 = x + (w - larg_tab) / 2
        n = len(linhas) + 1
        gf = self.sl.shapes.add_table(n, len(cab), Cm(x0), Cm(y + 1.3), Cm(larg_tab),
                                      Cm(alt_linha * n))
        tbl = gf.table
        # "No Style, Table Grid": bordas finas pretas, sem as cores do tema.
        tbl._tbl.tblPr.find(
            "{http://schemas.openxmlformats.org/drawingml/2006/main}tableStyleId").text = \
            "{5940675A-B579-460E-94D1-54222C63F5DA}"
        tbl.first_row = True
        for j, lw in enumerate(larguras):
            tbl.columns[j].width = Cm(lw)
        for i, linha in enumerate([cab] + linhas):
            tbl.rows[i].height = Cm(alt_linha)
            for j, val in enumerate(linha):
                cel = tbl.cell(i, j)
                cel.fill.solid()
                cel.fill.fore_color.rgb = CINZA if i == 0 else RGBColor(0xFF, 0xFF, 0xFF)
                cel.margin_left = cel.margin_right = Cm(0.15)
                cel.margin_top = cel.margin_bottom = Cm(0.03)
                cel.vertical_anchor = MSO_ANCHOR.MIDDLE
                tf = cel.text_frame
                tf.paragraphs[0].alignment = PP_ALIGN.LEFT if j == 0 else PP_ALIGN.CENTER
                negrito = i == 0 or (negrito_ultima and i == n - 1)
                runs_ricos(tf.paragraphs[0], val, CORPO, negrito)
        fim = y + 1.3 + alt_linha * n
        self.fonte(x, fim + 0.1, w)
        return fim + 1.1

    def gerar(self):
        for nome in REMOVER:
            sh = self.por_nome[nome]
            sh._element.getparent().remove(sh._element)

        # Cabeçalho
        trocar_texto(self.por_nome["CaixaDeTexto 1"], [IDENT])
        trocar_texto(self.por_nome["CaixaDeTexto 55"], [TITULO], tam=54)
        self.por_nome["CaixaDeTexto 55"].top = Cm(8.6)
        trocar_texto(self.por_nome["CaixaDeTexto 56"], NOMES[0])
        trocar_texto(self.por_nome["CaixaDeTexto 57"], NOMES[1])
        orient = self.por_nome["CaixaDeTexto 58"]
        trocar_texto(orient, [ORIENTADORES])
        orient.left, orient.width = Cm(1.0), Cm(57.4)

        # INTRODUÇÃO (esquerda, cabeçalho em 23,4 cm)
        self.texto(ESQ, 26.0, LARG, 11.0, [INTRODUCAO])
        self.figura(ESQ, 37.6, LARG, "Figura 1", "Arquitetura da bancada simulada.",
                    FIG / "arquitetura.png", 26.0)

        # Modelo/Específico (esquerda, cabeçalho em 56,1 cm)
        self.texto(ESQ, 58.6, LARG, 3.2, [MODELO_INTRO])
        self.texto(ESQ, 61.9, LARG, 4.4, EQUACOES, PP_ALIGN.CENTER)
        self.texto(ESQ, 66.8, LARG, 8.0, [TAREFAS])

        # IMPL. E RESULTADOS (direita, cabeçalho em 23,4 cm)
        self.texto(DIR, 26.0, LARG, 8.0, [RESULTADOS])
        y = self.figura(DIR, 33.4, LARG, "Figura 2", "Resposta a −20 % na vazão do bico (2000 rpm).",
                        FIG / "degrau.png", 20.0)
        larg_tab = 15.4
        fim_tab = self.tabela(DIR, y + 0.4, larg_tab, "Tabela 2", "Metas e valores medidos.",
                    ["Métrica", "Meta", "Medido"], [list(m) for m in METAS], [6.6, 4.4, 4.4])
        x3 = DIR + larg_tab + 0.4
        fim_fig = self.figura(x3, y + 0.4, LARG - larg_tab - 0.4, "Figura 3", "Painel do operador.",
                    RAIZ / "docs" / "painel.png", LARG - larg_tab - 0.6)

        # CONCLUSÃO (direita)
        conc = self.por_nome["CaixaDeTexto 40"]
        topo = max(fim_tab, fim_fig) + 0.6
        conc.top = Cm(topo)
        self.texto(DIR, topo + 2.6, LARG, 6.0, [CONCLUSAO])

        # Custo de desenvolvimento, abaixo da conclusão
        total_h = sum(h for _, h in CUSTO)
        linhas = [[e, str(h), reais(h * 50)] for e, h in CUSTO]
        linhas.append(["Total", str(total_h), reais(total_h * 50)])
        self.tabela(DIR, topo + 8.9, LARG, "Tabela 1",
                    "Custo de desenvolvimento (R$ 50/h; software livre, R$ 0).",
                    ["Etapa", "Horas", "Custo"], linhas, [8.0, 3.5, 5.5], negrito_ultima=True)

        saida = AQUI / "banner.pptx"
        self.prs.save(saida)
        subprocess.run(["soffice", "--headless", "--convert-to", "pdf", "--outdir", str(AQUI),
                        str(saida)], check=True, capture_output=True)
        print("banner.pptx e banner.pdf gerados")


if __name__ == "__main__":
    Banner().gerar()
