/*
 * Base de tempo do HAL (HAL_GetTick) no TIM14 a 1 kHz.
 * O SysTick fica exclusivo para o FreeRTOS, como no CubeMX.
 * TIM14 e não TIM16/17: no Renode o update desses dois está ligado ao DMA
 * (canais 1 e 2) mesmo sem UDE, o que gerava leituras extras do ADC.
 */
#include "stm32f0xx_hal.h"
#include "board.h"

static TIM_HandleTypeDef htim14;

HAL_StatusTypeDef HAL_InitTick(uint32_t TickPriority)
{
    __HAL_RCC_TIM14_CLK_ENABLE();

    htim14.Instance = TIM14;
    htim14.Init.Prescaler = TIMER_PRESCALER;          /* 1 MHz */
    htim14.Init.CounterMode = TIM_COUNTERMODE_UP;
    htim14.Init.Period = (TIMER_TICK_HZ / 1000u) - 1u; /* 1 ms */
    htim14.Init.ClockDivision = TIM_CLOCKDIVISION_DIV1;
    htim14.Init.RepetitionCounter = 0;
    htim14.Init.AutoReloadPreload = TIM_AUTORELOAD_PRELOAD_DISABLE;
    if(HAL_TIM_Base_Init(&htim14) != HAL_OK) {
        return HAL_ERROR;
    }

    HAL_NVIC_SetPriority(TIM14_IRQn, TickPriority, 0);
    HAL_NVIC_EnableIRQ(TIM14_IRQn);
    uwTickPrio = TickPriority;
    return HAL_TIM_Base_Start_IT(&htim14);
}

void HAL_SuspendTick(void)
{
    __HAL_TIM_DISABLE_IT(&htim14, TIM_IT_UPDATE);
}

void HAL_ResumeTick(void)
{
    __HAL_TIM_ENABLE_IT(&htim14, TIM_IT_UPDATE);
}

void TIM14_IRQHandler(void)
{
    if(__HAL_TIM_GET_FLAG(&htim14, TIM_FLAG_UPDATE)) {
        __HAL_TIM_CLEAR_FLAG(&htim14, TIM_FLAG_UPDATE);
        HAL_IncTick();
    }
}
