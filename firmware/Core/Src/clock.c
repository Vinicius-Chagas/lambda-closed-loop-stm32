/*
 * Relógio do sistema em 48 MHz.
 *
 * Feito com registradores em vez de HAL_RCC_ClockConfig() porque o HAL
 * relê FLASH->ACR e os campos PLLMUL para validar a configuração, e esses
 * campos não são modelados pelo Renode. Em hardware real o resultado é o
 * mesmo.
 */
#include "stm32f0xx.h"
#include "ecu_hw.h"

#define CLOCK_TIMEOUT 100000u

static bool wait_flag(volatile uint32_t *reg, uint32_t mask, uint32_t expected)
{
    for(uint32_t i = 0; i < CLOCK_TIMEOUT; i++) {
        if((*reg & mask) == expected) {
            return true;
        }
    }
    return false;
}

static bool pll_from_hse(void)
{
    RCC->CR |= RCC_CR_HSEON;
    if(!wait_flag(&RCC->CR, RCC_CR_HSERDY, RCC_CR_HSERDY)) {
        RCC->CR &= ~RCC_CR_HSEON;
        return false;
    }
    /* PLL = HSE (8 MHz) / 1 x 6 = 48 MHz */
    RCC->CFGR2 = RCC_CFGR2_PREDIV_DIV1;
    RCC->CFGR = (RCC->CFGR & ~(RCC_CFGR_PLLSRC | RCC_CFGR_PLLMUL))
                | RCC_CFGR_PLLSRC_HSE_PREDIV | RCC_CFGR_PLLMUL6;
    RCC->CR |= RCC_CR_PLLON;
    if(!wait_flag(&RCC->CR, RCC_CR_PLLRDY, RCC_CR_PLLRDY)) {
        return false;
    }
    RCC->CFGR = (RCC->CFGR & ~RCC_CFGR_SW) | RCC_CFGR_SW_PLL;
    return wait_flag(&RCC->CFGR, RCC_CFGR_SWS, RCC_CFGR_SWS_PLL);
}

static bool hsi48(void)
{
    RCC->CR2 |= RCC_CR2_HSI48ON;
    if(!wait_flag(&RCC->CR2, RCC_CR2_HSI48RDY, RCC_CR2_HSI48RDY)) {
        return false;
    }
    RCC->CFGR = (RCC->CFGR & ~RCC_CFGR_SW) | RCC_CFGR_SW_HSI48;
    return wait_flag(&RCC->CFGR, RCC_CFGR_SWS, RCC_CFGR_SWS_HSI48);
}

void clock_init(void)
{
    /* 48 MHz exige 1 wait state na flash. */
    FLASH->ACR = FLASH_ACR_PRFTBE | FLASH_ACR_LATENCY;

    if(!pll_from_hse() && !hsi48()) {
        /* Sem fonte de 48 MHz: segue no HSI de 8 MHz, mas as constantes de
         * tempo ficariam erradas, então é melhor parar. */
        for(;;) {}
    }
    SystemCoreClock = SYSCLK_HZ;
}
