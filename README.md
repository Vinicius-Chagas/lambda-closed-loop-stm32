# Controle em malha fechada da mistura ar-combustível (STM32 + Renode)

Bancada simulada de uma ECU que controla a mistura ar-combustível a partir da sonda lambda.

- **ECU**: firmware C (HAL STM32F0 + FreeRTOS) para STM32F072, emulado no [Renode](https://renode.io).
- **Planta**: motor, escape e sonda simulados em Go, mestre do tempo da simulação.
- **Painel**: Go + raylib, uma tela, lendo o barramento CAN (`vcan0`).

Física e parâmetros: [`docs/equacoes.md`](docs/equacoes.md). Andamento: [`docs/PROGRESSO.md`](docs/PROGRESSO.md).

## Requisitos (Linux)

| | Arch | Fedora | Debian/Ubuntu |
|---|---|---|---|
| Toolchain ARM | `arm-none-eabi-gcc arm-none-eabi-newlib` | `arm-none-eabi-gcc-cs arm-none-eabi-newlib` | `gcc-arm-none-eabi libnewlib-arm-none-eabi` |
| Build | `cmake make` | `cmake make` | `cmake make` |
| Renode | Docker (imagem fixada no `Makefile`) | idem | idem |
| CAN | `can-utils` | `can-utils` | `can-utils` |

## Comandos

```sh
make firmware          # compila firmware/build/ecu.elf
make test-unit         # Unity (controle C) + go test (planta, SIL)
make test-sil          # malha fechada C × planta Go, com as medições de acomodação
make test-integration  # SIL + testes Robot no Renode + cliente Go contra o Renode
make renode            # Renode interativo com a ECU carregada
./scripts/setup-vcan.sh   # cria vcan0 (root)
```
