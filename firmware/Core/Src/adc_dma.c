/*
 * Leitura dos sensores: ADC1 em varredura (IN0..IN2) disparada pelo TIM15
 * a 1 kHz, com DMA circular. O buffer tem dois blocos de 10 varreduras;
 * enquanto o DMA preenche um bloco, a ControlTask lê o outro.
 */
#include <string.h>

#include "stm32f0xx_hal.h"
#include "ecu_hw.h"

#define BLOCK_LEN (ADC_SETS_PER_BLOCK * ADC_NUM_CHANNELS)

static ADC_HandleTypeDef hadc;
static DMA_HandleTypeDef hdma_adc;
static TIM_HandleTypeDef htim15;

static uint16_t dma_buffer[2 * BLOCK_LEN];
static volatile const uint16_t *ready_block;
static volatile uint32_t ready_timestamp_us;
static TaskHandle_t notify_task;

static void trigger_timer_init(void)
{
    __HAL_RCC_TIM15_CLK_ENABLE();
    htim15.Instance = TIM15;
    htim15.Init.Prescaler = TIMER_PRESCALER;
    htim15.Init.CounterMode = TIM_COUNTERMODE_UP;
    htim15.Init.Period = (TIMER_TICK_HZ / ADC_SAMPLE_RATE_HZ) - 1u;
    htim15.Init.ClockDivision = TIM_CLOCKDIVISION_DIV1;
    htim15.Init.RepetitionCounter = 0;
    htim15.Init.AutoReloadPreload = TIM_AUTORELOAD_PRELOAD_ENABLE;
    if(HAL_TIM_Base_Init(&htim15) != HAL_OK) {
        ecu_fatal("TIM15 init");
    }
    TIM_MasterConfigTypeDef master = {
        .MasterOutputTrigger = TIM_TRGO_UPDATE,
        .MasterSlaveMode = TIM_MASTERSLAVEMODE_DISABLE,
    };
    HAL_TIMEx_MasterConfigSynchronization(&htim15, &master);
}

void adc_dma_init(TaskHandle_t notify)
{
    notify_task = notify;

    __HAL_RCC_GPIOA_CLK_ENABLE();
    GPIO_InitTypeDef gpio = {
        .Pin = GPIO_PIN_0 | GPIO_PIN_1 | GPIO_PIN_2,
        .Mode = GPIO_MODE_ANALOG,
        .Pull = GPIO_NOPULL,
    };
    HAL_GPIO_Init(GPIOA, &gpio);

    __HAL_RCC_DMA1_CLK_ENABLE();
    hdma_adc.Instance = DMA1_Channel1;
    hdma_adc.Init.Direction = DMA_PERIPH_TO_MEMORY;
    hdma_adc.Init.PeriphInc = DMA_PINC_DISABLE;
    hdma_adc.Init.MemInc = DMA_MINC_ENABLE;
    hdma_adc.Init.PeriphDataAlignment = DMA_PDATAALIGN_HALFWORD;
    hdma_adc.Init.MemDataAlignment = DMA_MDATAALIGN_HALFWORD;
    hdma_adc.Init.Mode = DMA_CIRCULAR;
    hdma_adc.Init.Priority = DMA_PRIORITY_HIGH;
    if(HAL_DMA_Init(&hdma_adc) != HAL_OK) {
        ecu_fatal("DMA init");
    }
    __HAL_LINKDMA(&hadc, DMA_Handle, hdma_adc);
    HAL_NVIC_SetPriority(DMA1_Channel1_IRQn, 1, 0);
    HAL_NVIC_EnableIRQ(DMA1_Channel1_IRQn);

    __HAL_RCC_ADC1_CLK_ENABLE();
    hadc.Instance = ADC1;
    hadc.Init.ClockPrescaler = ADC_CLOCK_SYNC_PCLK_DIV4;
    hadc.Init.Resolution = ADC_RESOLUTION_12B;
    hadc.Init.DataAlign = ADC_DATAALIGN_RIGHT;
    hadc.Init.ScanConvMode = ADC_SCAN_DIRECTION_FORWARD;
    hadc.Init.EOCSelection = ADC_EOC_SEQ_CONV;
    hadc.Init.LowPowerAutoWait = DISABLE;
    hadc.Init.LowPowerAutoPowerOff = DISABLE;
    hadc.Init.ContinuousConvMode = DISABLE;
    hadc.Init.DiscontinuousConvMode = DISABLE;
    /* O gatilho externo (TIM15) é ligado depois do HAL_ADC_Init: o HAL relê
     * CFGR1 para validar a configuração e o modelo do Renode não guarda o
     * campo EXTSEL, o que faria o init falhar. No hardware é equivalente. */
    hadc.Init.ExternalTrigConv = ADC_SOFTWARE_START;
    hadc.Init.ExternalTrigConvEdge = ADC_EXTERNALTRIGCONVEDGE_NONE;
    hadc.Init.DMAContinuousRequests = ENABLE;
    hadc.Init.Overrun = ADC_OVR_DATA_OVERWRITTEN;
    if(HAL_ADC_Init(&hadc) != HAL_OK) {
        ecu_fatal("ADC init");
    }

    static const uint32_t channels[ADC_NUM_CHANNELS] = {
        ADC_CHANNEL_0, ADC_CHANNEL_1, ADC_CHANNEL_2,
    };
    for(uint32_t i = 0; i < ADC_NUM_CHANNELS; i++) {
        ADC_ChannelConfTypeDef ch = {
            .Channel = channels[i],
            .Rank = ADC_RANK_CHANNEL_NUMBER,
            .SamplingTime = ADC_SAMPLETIME_71CYCLES_5,
        };
        if(HAL_ADC_ConfigChannel(&hadc, &ch) != HAL_OK) {
            ecu_fatal("ADC channel");
        }
    }

    if(HAL_ADCEx_Calibration_Start(&hadc) != HAL_OK) {
        ecu_fatal("ADC calibration");
    }

    MODIFY_REG(ADC1->CFGR1, ADC_CFGR1_EXTSEL | ADC_CFGR1_EXTEN,
               ADC_EXTERNALTRIGCONV_T15_TRGO | ADC_EXTERNALTRIGCONVEDGE_RISING);

    trigger_timer_init();
    if(HAL_ADC_Start_DMA(&hadc, (uint32_t *)dma_buffer, 2 * BLOCK_LEN) != HAL_OK) {
        ecu_fatal("ADC start");
    }
    HAL_TIM_Base_Start(&htim15);
}

static void block_ready(const uint16_t *block)
{
    ready_block = block;
    ready_timestamp_us = timing_now_us();
    if(notify_task != NULL) {
        BaseType_t woken = pdFALSE;
        vTaskNotifyGiveFromISR(notify_task, &woken);
        portYIELD_FROM_ISR(woken);
    }
}

/*
 * Tratador próprio em vez de HAL_DMA_IRQHandler. A metade pronta é deduzida
 * do contador restante (CNDTR): se o DMA está escrevendo a 2ª metade, a 1ª
 * acabou de ficar pronta, e vice-versa. Assim não dependemos das flags HT/TC
 * nem lemos CCR na interrupção - o modelo de DMA do Renode desliga o canal
 * quando CCR é lido e repete a flag HT a cada transferência.
 */
void DMA1_Channel1_IRQHandler(void)
{
    static const uint16_t *last_notified;

    DMA1->IFCR = DMA_IFCR_CGIF1;
    const uint16_t *done = (DMA1_Channel1->CNDTR <= BLOCK_LEN)
                         ? &dma_buffer[0] : &dma_buffer[BLOCK_LEN];
    if(done != last_notified) {
        last_notified = done;
        block_ready(done);
    }
}

void adc_dma_block_average(uint16_t out[ADC_NUM_CHANNELS])
{
    uint32_t sum[ADC_NUM_CHANNELS] = {0};
    const uint16_t *block = (const uint16_t *)ready_block;
    if(block == NULL) {
        memset(out, 0, ADC_NUM_CHANNELS * sizeof(out[0]));
        return;
    }
    for(uint32_t s = 0; s < ADC_SETS_PER_BLOCK; s++) {
        for(uint32_t c = 0; c < ADC_NUM_CHANNELS; c++) {
            sum[c] += block[s * ADC_NUM_CHANNELS + c];
        }
    }
    for(uint32_t c = 0; c < ADC_NUM_CHANNELS; c++) {
        out[c] = (uint16_t)((sum[c] + ADC_SETS_PER_BLOCK / 2u) / ADC_SETS_PER_BLOCK);
    }
}

uint32_t adc_dma_block_timestamp_us(void)
{
    return ready_timestamp_us;
}
