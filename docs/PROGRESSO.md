# Progresso

Registro de cada etapa: o que foi feito, como foi verificado, valores medidos e desvios do plano.

| Etapa | Status |
|---|---|
| 0. Prova de conceito | ✅ local e CI |
| 1. Equações | ✅ |
| 2. Periféricos | ✅ local e CI |
| 3. Planta (Go) | ✅ local |
| 4. Controle | ✅ local |
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
| CI `vcan` (`scripts/check-vcan-bridge.sh`) | ✅ 20 frames 0x100 da ECU no `vcan0` do runner. Na 1ª tentativa só 3 chegavam: o container subia sem `-i`, o console do Renode recebia EOF e a emulação encerrava |

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

## Etapa 3 — Planta (Go)

**Feito** (`go/`, módulo único)
- `internal/engine`: motor (rotação de 1ª ordem seguindo a borboleta, ar alfa-N, combustível do bico
  com tempo morto, λ no cilindro) e atraso de transporte do escape numa linha de atraso com
  interpolação. Parâmetros em `params.go`, iguais a `docs/equacoes.md` e `firmware/control/params.h`.
- `internal/sensors`: sonda (1ª ordem, τ = 100 ms), TPS e conversor F/V; **uma goroutine por sensor**,
  sincronizadas por passo numa barreira (`Bank`). O relógio é o da simulação.
- `internal/renode`: cliente do monitor (TCP, trata a negociação telnet, eco e prompt, reconexão).
  Cada passo é **um comando** `lockstep <V λ> <V tps> <V rpm> <dt>` (`renode/lockstep.py`, Python
  dentro do Renode), que ajusta o ADC, roda `dt` e devolve CCR1/ARR do TIM2.
- `internal/plant`: laço da bancada (física em subpassos de 1 ms, sincronização com a ECU a cada
  5 ms), comandos do painel por canal, gravação CSV. A ECU é uma interface: Renode agora, controle C
  via cgo no SIL (etapa 4).
- `internal/canbus`: codec dos frames (mesmo vetor de teste do firmware) e SocketCAN
  (`go.einride.tech/can`).
- `cmd/planta`: liga tudo; recebe 0x310 do painel e publica 0x300 a cada 20 ms simulados.
- `renode/lambda.resc` agora usa `$ORIGIN`: funciona a partir de qualquer diretório e já carrega os
  comandos `lockstep` e `can_send`.

**Verificação**

| Comando | Resultado |
|---|---|
| `make test-unit-go` (`go vet` + `go test -race ./...`) | OK: curvas e atraso (tabela de `equacoes.md`), degrau do bico aparece no escape após `Td` (±3 ms), sonda com 63,2 % em τ, barreira sem passos pulados (4 sensores × 1000 passos, sem corrida), codec, cliente do monitor contra servidor falso, laço da planta e CSV |
| `make test-renode-go` | OK contra o Renode real: período do TIM2 = 120/RPM em 7273 rpm e em marcha lenta |
| `planta -script @renode/lambda.resc -can "" -duracao 3` | roda com o firmware no Renode; CSV gerado |

Velocidade do lockstep (passo de 5 ms): 70 passos/s com 6 comandos por passo → **96 passos/s
(0,48× o tempo real)** com o comando único `lockstep`. A emulação livre da ECU anda a ~1,0× (medido
com `RunFor 5`), então esse é o teto. Para a feira, `-sync 10` dá ~0,7×.

**Desvios**
- O Renode não reenvia o prompt numa reconexão ao monitor; o cliente manda uma linha vazia e descarta
  os prompts acumulados.

## Etapa 4 — Controle

**Feito** (`firmware/control/`, C puro, o mesmo código no firmware, nos testes Unity e no SIL)
- `fuel_map.c`: tabela de combustível base 7 (TPS) × 8 (RPM), interpolação bilinear, borda sem
  extrapolar. Os pontos saem do modelo nominal de `docs/equacoes.md` (o teste confere cada um).
- `controller.c`: `t_i = t_morto + t_base·(1 + c)`, PI sobre `e = λ_med − λ_alvo` (Kp 0,15,
  Ki 2,0/s, Δt 10 ms) com anti-windup condicional; malha aberta zera o integrador; motor parado
  (< 100 rpm) não injeta; pulso limitado a 90 % do ciclo e a 60 ms.
- `ecu.c`: um período de controle a partir das contagens do ADC (`ecu_step`) e comandos do painel
  validados (`ecu_set_alvo`, `ecu_set_modo`).
- `ControlTask` usa `ecu_step`; a correção vai no frame 0x100.
- SIL (`go/internal/sil`): o cgo compila `firmware/control/*.c` (um `.c` de inclusão por fonte) e
  a ECU resultante implementa `plant.ECU`: soma as amostras de 10 ms e roda o controle, como o
  firmware. `plant.AnalisarDegrau` mede desvio, sobressinal e acomodação (reusado no e2e).

**Verificação**

| Comando | Resultado |
|---|---|
| `make test-unit-c` | 5/5 suítes Unity: tabela = modelo em todos os 56 pontos (±1 µs), interpolação, bordas, monotonia; PI: sinal da correção, 1º passo = (Kp + Ki·Δt)·e, saturação e anti-windup, malha aberta zera a integral, motor parado, limite de 90 % do ciclo, comandos inválidos ignorados |
| `make test-unit-go` | inclui o SIL e `AnalisarDegrau` |
| `make test-renode` | 7/7: o injetor agora vem do controle (malha aberta em 2500 rpm/20 % = modelo ±0,5 %); novo teste da malha fechada no Renode: sonda pobre leva o pulso a `t_morto + 1,30·t_base`, sonda rica a `0,70·t_base`, malha aberta volta à tabela; ti do frame 0x100 = CCR1 do TIM2 |

Resposta em malha fechada (SIL, 2000 rpm, perturbação aplicada com λ = 1 estável; λ medido na
sonda; banda ±0,02):

| Perturbação | Desvio máx. | Sobressinal | Acomodação | λ final | Correção final |
|---|---|---|---|---|---|
| Bico −20 % (vazão 0,8) | 0,203 | 0,0 % | 1,31 s | 1,0009 | +25,4 % |
| Bico +20 % (vazão 1,2) | 0,139 | 0,1 % | 1,09 s | 1,0000 | −16,3 % |
| Vazamento de ar 10 mg | 0,069 | 0,1 % | 0,88 s | 1,0002 | +8,8 % |
| Bico −20 % em marcha lenta (800 rpm, pior atraso) | 0,234 | 2,9 % | 1,18 s | 0,9996 | — |
| Bico −20 % em malha aberta | λ fica em ≈ 1,25 | | | | 0 |

Metas: acomodação ≤ 2 s ✅, sobressinal ≤ 5 % ✅.

Tempos da `ControlTask` no Renode com o controle: período 10 000 µs, jitter 10 µs, execução máx.
**158 µs** (era 71 µs com o pulso fixo; o M0 não tem FPU e o PI e a tabela usam `float` por
software), 0 deadlines perdidos.

**Desvios**
- **Limite da correção de ±25 % para ±30 %.** O bico com 80 % da vazão pede exatamente +25 %; com
  ±25 % o PI ficava encostado no limite e o erro da tabela sobrava em λ (λ final 1,004). Com ±30 %
  a correção assenta em +25,4 % com folga (`docs/equacoes.md` e `can/lambda.dbc` atualizados).
- **Anti-windup**: a forma "não integra se saturar" deixava a saída parada em 0,249, um passo antes
  do limite (achado pelo teste Unity). Agora a integral anda só até a saída encostar no limite.
- O controle não usa `libm` no alvo (sem `fminf`/`fmaxf`), para não aumentar a FLASH.

## Pendências para a equipe

Nenhuma até agora.
