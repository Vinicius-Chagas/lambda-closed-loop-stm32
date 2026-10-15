/*
 * Relógio de medição em µs. O Cortex-M0 não tem o contador de ciclos DWT,
 * então usamos o TIM7 (16 bits a 1 MHz) estendido para 32 bits contando os
 * estouros na interrupção.
 */
#include "stm32f0xx.h"
#include "ecu_hw.h"

static volatile uint32_t overflow_high;

void timing_init(void)
{
    RCC->APB1ENR |= RCC_APB1ENR_TIM7EN;
    TIM7->CR1 = 0;
    TIM7->PSC = TIMER_PRESCALER;
    TIM7->ARR = 0xFFFFu;
    TIM7->CNT = 0;
    TIM7->EGR = TIM_EGR_UG;   /* carrega o PSC */
    TIM7->SR = 0;
    TIM7->DIER = TIM_DIER_UIE;
    NVIC_SetPriority(TIM7_IRQn, 0);
    NVIC_EnableIRQ(TIM7_IRQn);
    TIM7->CR1 = TIM_CR1_CEN;
}

void TIM7_IRQHandler(void)
{
    if(TIM7->SR & TIM_SR_UIF) {
        TIM7->SR = ~TIM_SR_UIF;
        overflow_high += 0x10000u;
    }
}

uint32_t timing_now_us(void)
{
    uint32_t primask = __get_PRIMASK();
    __disable_irq();
    uint32_t high = overflow_high;
    uint32_t low = TIM7->CNT;
    /* Estouro pendente ainda não tratado: o contador já voltou a zero. */
    if((TIM7->SR & TIM_SR_UIF) && low < 0x8000u) {
        high += 0x10000u;
    }
    __set_PRIMASK(primask);
    return high + low;
}
