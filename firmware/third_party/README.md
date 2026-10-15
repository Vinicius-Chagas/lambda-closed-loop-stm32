# Bibliotecas de terceiros (vendorizadas)

Copiadas sem modificação dos repositórios oficiais para que o projeto compile sem submódulos.
Cada pasta mantém o arquivo de licença original.

| Pasta | Origem | Commit |
|---|---|---|
| `cmsis_core` | github.com/STMicroelectronics/cmsis_core (`CMSIS/Core/Include`, só Cortex-M0) | 01d1814 |
| `cmsis_device_f0` | github.com/STMicroelectronics/cmsis_device_f0 (só STM32F072xB) | 1ed05aa |
| `stm32f0xx_hal_driver` | github.com/STMicroelectronics/stm32f0xx_hal_driver v1.7.9 (módulos usados) | 524eba1 |
| `FreeRTOS-Kernel` | github.com/FreeRTOS/FreeRTOS-Kernel (porta GCC/ARM_CM0, heap_4) | 8be86d4 |
| `Unity` | github.com/ThrowTheSwitch/Unity | 2b80d1a |

Para adicionar um módulo do HAL (ex.: SPI), copie `stm32f0xx_hal_spi.c` do repositório de origem
para `stm32f0xx_hal_driver/Src/` e habilite `HAL_SPI_MODULE_ENABLED` em `Core/Inc/stm32f0xx_hal_conf.h`.
