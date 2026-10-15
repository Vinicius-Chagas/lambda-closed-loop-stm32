/*
 * Bico injetor no TIM2 (32 bits, 1 µs por contagem), canal 1 em PWM:
 * período = um ciclo do motor (2 voltas), largura = tempo de injeção.
 * O TIM2 é usado porque em marcha lenta o ciclo passa de 65 ms, o limite
 * de um timer de 16 bits contando em µs.
 */
#include "stm32f0xx_hal.h"
#include "ecu_hw.h"

static TIM_HandleTypeDef htim2;

void injector_init(void)
{
    __HAL_RCC_GPIOA_CLK_ENABLE();
    GPIO_InitTypeDef gpio = {
        .Pin = GPIO_PIN_5,
        .Mode = GPIO_MODE_AF_PP,
        .Pull = GPIO_NOPULL,
        .Speed = GPIO_SPEED_FREQ_HIGH,
        .Alternate = GPIO_AF2_TIM2,
    };
    HAL_GPIO_Init(GPIOA, &gpio);

    __HAL_RCC_TIM2_CLK_ENABLE();
    htim2.Instance = TIM2;
    htim2.Init.Prescaler = TIMER_PRESCALER;
    htim2.Init.CounterMode = TIM_COUNTERMODE_UP;
    htim2.Init.Period = 100000u - 1u;                   /* 100 ms até a 1ª atualização */
    htim2.Init.ClockDivision = TIM_CLOCKDIVISION_DIV1;
    htim2.Init.AutoReloadPreload = TIM_AUTORELOAD_PRELOAD_ENABLE;
    if(HAL_TIM_PWM_Init(&htim2) != HAL_OK) {
        ecu_fatal("TIM2 init");
    }
    TIM_OC_InitTypeDef oc = {
        .OCMode = TIM_OCMODE_PWM1,
        .Pulse = 0,
        .OCPolarity = TIM_OCPOLARITY_HIGH,
        .OCFastMode = TIM_OCFAST_DISABLE,
    };
    if(HAL_TIM_PWM_ConfigChannel(&htim2, &oc, TIM_CHANNEL_1) != HAL_OK) {
        ecu_fatal("TIM2 channel");
    }
    HAL_TIM_PWM_Start(&htim2, TIM_CHANNEL_1);
}

void injector_set(uint32_t cycle_period_us, uint32_t pulse_us)
{
    if(cycle_period_us < 2u) {
        cycle_period_us = 2u;
    }
    if(pulse_us >= cycle_period_us) {
        pulse_us = cycle_period_us - 1u;
    }
    /* Com preload, os novos valores valem a partir do próximo ciclo. */
    __HAL_TIM_SET_AUTORELOAD(&htim2, cycle_period_us - 1u);
    __HAL_TIM_SET_COMPARE(&htim2, TIM_CHANNEL_1, pulse_us);
}

uint32_t injector_pulse_us(void)
{
    return __HAL_TIM_GET_COMPARE(&htim2, TIM_CHANNEL_1);
}
