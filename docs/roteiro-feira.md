# Roteiro da feira

## Antes de abrir o estande

1. `make firmware` e `./scripts/setup-vcan.sh` (o `vcan0` some quando a máquina reinicia).
2. Terminal 1: `make run`. Esperar o log da planta mostrar `t=… RPM= 800 λsonda=1.000`.
3. Terminal 2: `make painel`, depois **F** para tela cheia.
4. Conferir no painel: selo **MALHA FECHADA**, λ ≈ 1,000 em verde, linha de tempos da ControlTask
   com jitter ≤ 100 µs e 0 deadlines perdidos.
5. Plano B (sem Renode/Docker): `make painel-demo` — mesmo controle C, rodando dentro do painel.

A simulação anda a ~0,5× o tempo real com a ECU no Renode (o Renode emula o STM32 instrução a
instrução e a planta conversa com ele a cada 5 ms). No modo demo anda em tempo real.

## Demonstração (3 a 5 minutos)

| Passo | Tecla | O que mostrar / falar |
|---|---|---|
| 1 | — | "λ = 1 é a mistura ideal: 14,7 g de ar para cada 1 g de gasolina. Abaixo de 1 a mistura é **rica** (sobra combustível), acima é **pobre** (sobra ar)." |
| 2 | ↑ ↑ ↑ ↑ | Acelerar até ~20 %. A rotação sobe, o tempo de injeção sobe junto (mais ar → mais combustível). λ quase não sai do 1: a tabela da ECU já prevê o combustível. |
| 3 | 1 | **Bico −20 %** (simula um bico entupido). No gráfico: a linha cinza (λ no cilindro) pula na hora; a amarela (sonda) só reage ~0,1 s depois — é o **atraso do escape** até a sonda. A ECU vê a mistura pobre e aumenta a **correção** até ~+25 %; λ volta a 1,00 em menos de 2 s. |
| 4 | Espaço | **Malha aberta**: a ECU ignora a sonda. A correção vai a 0 e λ fica em ~1,25 (pobre) — é assim que um motor sem sonda lambda ficaria. |
| 5 | Espaço | Malha fechada de novo: λ volta a 1. |
| 6 | 1, depois 2 | Tira o defeito do bico e liga o **vazamento de ar** (ar falso depois do sensor). Mesma história: a sonda percebe e o controle compensa (~+9 %). |
| 7 | 2, ↓ ↓ ↓ ↓ | Volta tudo ao normal e à marcha lenta. |

Perguntas comuns:
- **Por que não corrige na hora?** O gás leva tempo para chegar à sonda (atraso de transporte,
  maior em rotação baixa) e a sonda tem resposta de ~0,1 s. Um controle rápido demais oscilaria.
- **Isso é tempo real?** A tarefa de controle da ECU roda a cada 10 ms; o painel mostra o jitter
  (variação do período) e o tempo de execução medidos dentro do firmware, no tempo do processador
  emulado.
- **Onde está o microcontrolador?** Emulado no Renode: é o mesmo `.elf` que iria gravado num
  STM32F072 de verdade (ADC com DMA, timer gerando o pulso do bico, CAN, FreeRTOS).
- **Por que CAN?** É o barramento usado nos carros; o painel e a planta conversam com a ECU como
  um equipamento de diagnóstico conversaria (`candump vcan0` mostra os frames).

## Checklist do painel (verificação manual)

- [ ] Abre em 1280×720 e em tela cheia (F) sem cortar nada.
- [ ] Sem a bancada: "SEM SINAL DA BANCADA" no cabeçalho e "—" nos valores.
- [ ] ↑/↓ mudam o acelerador de 5 em 5 % (0 a 100).
- [ ] 1 liga/desliga "Vazão do bico 80 %" (em laranja quando perturbado).
- [ ] 2 liga/desliga "Vazamento de ar 10 mg".
- [ ] Espaço alterna o selo MALHA FECHADA (verde) / MALHA ABERTA (laranja).
- [ ] Barra: marcador em RICA/IDEAL/POBRE conforme λ; o número muda de cor junto.
- [ ] Gráfico: 10 s, λ no cilindro (cinza), na sonda (amarelo), alvo (verde) com a faixa ±0,02.
- [ ] Linha de tempos da ControlTask preenchida (só com a ECU no Renode).
