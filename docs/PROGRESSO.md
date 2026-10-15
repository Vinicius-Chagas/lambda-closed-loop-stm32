# Progresso

Registro de cada etapa: o que foi feito, como foi verificado, valores medidos e desvios do plano.

| Etapa | Status |
|---|---|
| 0. Prova de conceito | ✅ local · CI: build e Robot verdes; ponte vcan0 em ajuste |
| 1. Equações | ✅ |
| 2. Periféricos | ✅ local |
| 3. Planta (Go) | ⏳ |
| 4. Controle | ⏳ |
| 5. Tempos + e2e | ⏳ |
| 6. Painel | ⏳ |

## Etapa 0 — Prova de conceito

**Feito**
- Toolchain: `arm-none-eabi-gcc` 13.2 + CMake; HAL STM32F0 v1.7.9, CMSIS, FreeRTOS-Kernel e Unity
  copiados em `firmware/third_party/` (fontes e commits em `firmware/third_party/README.md`).
- Firmware mínimo: clock 48 MHz, ADC de 3 canais por DMA disparado pelo TIM15 (1 kHz), pulso do
  injetor no TIM2, bxCAN 500 kbit/s, log na USART1, FreeRTOS com 1 tarefa notificada pelo DMA.
- Plataforma Renode `renode/stm32f072_lambda.repl` (+ `lambda.resc`, `lambda-vcan.resc`).
- Teste de integração `renode/tests/firmware.robot`.

**Verificação**

| Comando | Resultado |
|---|---|
| `make firmware` | compila sem avisos (`-Wall -Wextra -Werror`); FLASH 19,5 KB / 128 KB, RAM 9,6 KB / 16 KB |
| `make test-renode` | 4/4 testes OK: boot; ADC lê 1,65/0,5/3,0 V → 2048/620/3723 e depois 0,6/2,0/1,2 V; TIM2 ARR = 59999 e CCR1 = 1650; frame CAN 0x100 com os 3 valores do ADC |
| CI `vcan` (`scripts/check-vcan-bridge.sh`) | 1ª execução: a ponte funciona (frames `100#…` no `vcan0` do runner), mas só 3 frames chegaram antes de parar; diagnóstico em andamento (o script agora imprime o log do Renode) |

**Lockstep pelo monitor do Renode** (cliente Python provisório, TCP_NODELAY, 1 comando composto por
passo: `Volts; RunFor; ReadDoubleWord`):

| Passo Δt | Passos/s | Fator de tempo real |
|---|---|---|
| 1 ms | ~185 | 0,19× |
| 5 ms | ~105 | 0,53× |
| 10 ms | ~95 | 0,95× |

O custo é dominado pela ida e volta de cada comando no monitor (~5 ms), não pela emulação da CPU.
Decisão: a planta integra a física em passos internos de 1 ms e sincroniza com o Renode a cada
5 ms (padrão, configurável). Os testes de tempo usam o tempo virtual, então a velocidade não afeta
as medições.

**Desvios do plano e motivo**
- **Imagem do Renode nightly fixada por digest** (`antmicro/renode@sha256:b30df3a8…`, 1.17.0 de
  12/10/2026). A 1.16.1 (`latest`) não serve: RCC do F0 em Python que trava o clock, ADC sem DMA e
  sem `ADCChannelSource`.
- **Injetor no TIM2 (32 bits) em vez do TIM3.** Em marcha lenta o ciclo é de 150 ms; com 1 µs de
  resolução isso não cabe em 16 bits.
- **Base de tempo de medição no TIM7** (16 bits + contagem de estouros por software), já que o TIM2
  ficou com o injetor.
- **Tick do HAL no TIM14.** No Renode, o update do TIM16/TIM17 está ligado ao DMA (canais 1–4) mesmo
  sem o bit UDE; com o TIM17 cada tick gerava uma leitura extra do ADC no buffer.
- **Disparo do ADC configurado depois da calibração.** O modelo do ADC não guarda o campo EXTSEL e a
  verificação do `HAL_ADC_Init` falhava; a ECU inicia em disparo por software e grava EXTSEL/EXTEN
  direto no registrador.
- **Tratador próprio da interrupção do DMA.** No modelo, ler o CCR do canal desliga o bit EN (o
  `HAL_DMA_IRQHandler` lê o CCR e o DMA parava após a primeira metade). O tratador usa só ISR/IFCR e
  CNDTR para saber qual metade do buffer ficou pronta.
- **Clock configurado por registrador**, sem `HAL_RCC_OscConfig` (HSE→PLL ×6 com retorno ao HSI48).
- **CAN no Renode**: as quatro linhas de IRQ do bxCAN ligadas à NVIC (IRQ 30) no `.repl` e o bit
  SLEEP limpo antes do `HAL_CAN_Init` (renode/renode#998).
- **Idle do FreeRTOS com `WFI`**: o Renode pula o tempo ocioso (fator de tempo real 0,71 → 0,95 com
  passos de 10 ms).

## Etapa 1 — Equações

`docs/equacoes.md`: lambda, modelo alfa-N do ar, combustível pelo bico (vazão e tempo morto), atraso
de transporte `Td = 180/RPM + 0,020 s`, sonda banda larga de 1ª ordem (τ = 100 ms), escalas dos
sensores, cálculo do tempo de injeção (tabela + PI) e todos os parâmetros numéricos. Os mesmos valores
vão para `go/internal/engine/params.go` e `firmware/control/params.h` nas etapas 3 e 4.

## Etapa 2 — Periféricos e tarefas

**Feito**
- `ControlTask` (prioridade 3), acordada pelo DMA a cada 10 ms: média de 10 amostras por canal,
  conversão em λ/TPS/RPM (`control/sensors.c`), período do TIM2 = ciclo do motor (120/RPM),
  pulso provisório fixo de 3000 µs (o controle entra na etapa 4) e medição dos próprios tempos
  (`control/timing_stats.c`, base de 1 µs no TIM7).
- `CanTask` (prioridade 2), a cada 20 ms: frames 0x100 (estado) e 0x101 (sensores); 0x110 (tempos)
  a cada 100 ms; trata o comando 0x200 do painel (λ alvo, modo, zerar tempos); log na USART a cada 0,5 s.
- Frames definidos em `can/lambda.dbc` e codificados em `control/can_codec.c`.
- Testes unitários Unity no host (`firmware/tests`, `make test-unit`): conversões dos sensores, layout
  dos frames, estatísticas de tempo (inclusive estouro do contador de 32 bits).
- `renode/can_send.py`: comando de monitor `can_send <id> "<hex>"` para injetar frames na ECU nos testes.

**Verificação**

| Comando | Resultado |
|---|---|
| `make firmware` | sem avisos; FLASH 24,2 KB, RAM 9,7 KB |
| `make test-unit` | 3/3 suítes Unity OK |
| `make test-renode` | 6/6 OK: boot; ADC 3 canais via DMA; TIM2 com ciclo de 16,5 ms (7273 rpm) e 150 ms (marcha lenta, > 16 bits); frames 0x100/0x101 com os valores esperados; comando 0x200 aceito e valores fora da faixa ignorados; tempos da ControlTask |

Tempos da `ControlTask` medidos no Renode (2 s simulados após zerar):

| Métrica | Medido | Meta |
|---|---|---|
| Período | 10 000 µs | 10 ms |
| Jitter máximo | 10 µs | ≤ 100 µs |
| Execução máxima | 71 µs | ≤ 1 ms |
| Deadlines perdidos | 0 | 0 |

**Desvios**
- Estado e sensores em dois frames (0x100 e 0x101): os campos do plano não cabem em 8 bytes.
- Periféricos iniciados dentro da `ControlTask` (antes do escalonador o FreeRTOS mascara as
  interrupções e o `HAL_GetTick` fica parado).

## Pendências para a equipe

Nenhuma até agora.
