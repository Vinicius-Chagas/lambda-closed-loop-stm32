# Progresso

Registro de cada etapa: o que foi feito, como foi verificado, valores medidos e desvios do plano.

| Etapa | Status |
|---|---|
| 0. Prova de conceito | ✅ local · CI: ponte vcan0 aguardando |
| 1. Equações | ✅ |
| 2. Periféricos | ⏳ |
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
| CI `vcan` (`scripts/check-vcan-bridge.sh`) | aguardando a primeira execução (este container não tem SocketCAN) |

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

## Pendências para a equipe

Nenhuma até agora.
