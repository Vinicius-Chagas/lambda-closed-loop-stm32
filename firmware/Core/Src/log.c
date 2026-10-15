/*
 * Log de depuração na USART1 (PA9), 115200 8N1, transmissão bloqueante.
 * Usado só na inicialização e pela tarefa de monitoramento, nunca em ISR.
 */
#include <stdarg.h>
#include <stdio.h>

#include "stm32f0xx_hal.h"
#include "ecu_hw.h"

static UART_HandleTypeDef huart1;

void log_init(void)
{
    __HAL_RCC_GPIOA_CLK_ENABLE();
    GPIO_InitTypeDef gpio = {
        .Pin = GPIO_PIN_9 | GPIO_PIN_10,
        .Mode = GPIO_MODE_AF_PP,
        .Pull = GPIO_PULLUP,
        .Speed = GPIO_SPEED_FREQ_HIGH,
        .Alternate = GPIO_AF1_USART1,
    };
    HAL_GPIO_Init(GPIOA, &gpio);

    __HAL_RCC_USART1_CLK_ENABLE();
    huart1.Instance = USART1;
    huart1.Init.BaudRate = 115200;
    huart1.Init.WordLength = UART_WORDLENGTH_8B;
    huart1.Init.StopBits = UART_STOPBITS_1;
    huart1.Init.Parity = UART_PARITY_NONE;
    huart1.Init.Mode = UART_MODE_TX_RX;
    huart1.Init.HwFlowCtl = UART_HWCONTROL_NONE;
    huart1.Init.OverSampling = UART_OVERSAMPLING_16;
    huart1.Init.OneBitSampling = UART_ONE_BIT_SAMPLE_DISABLE;
    huart1.AdvancedInit.AdvFeatureInit = UART_ADVFEATURE_NO_INIT;
    HAL_UART_Init(&huart1);
}

void log_printf(const char *fmt, ...)
{
    char buf[96];
    va_list args;
    va_start(args, fmt);
    int n = vsnprintf(buf, sizeof(buf), fmt, args);
    va_end(args);
    if(n <= 0) {
        return;
    }
    if(n > (int)sizeof(buf) - 1) {
        n = sizeof(buf) - 1;
    }
    HAL_UART_Transmit(&huart1, (uint8_t *)buf, (uint16_t)n, 100);
}

void ecu_fatal(const char *what)
{
    log_printf("FATAL: %s\r\n", what);
    __disable_irq();
    for(;;) {}
}
