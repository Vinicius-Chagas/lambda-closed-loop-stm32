/*
 * Prova de conceito: uma tarefa acordada pelo DMA do ADC a cada 10 ms,
 * que copia a tensão da sonda para a largura do pulso do injetor e
 * publica as leituras no CAN (ID 0x100).
 */
#include "ecu_hw.h"
#include "app.h"

static TaskHandle_t control_task;

static void control_loop(void *arg)
{
    (void)arg;
    uint32_t blocks = 0;

    adc_dma_init(control_task);
    injector_init();
    if(!canbus_init()) {
        ecu_fatal("CAN init");
    }
    log_printf("ECU pronta\r\n");

    for(;;) {
        ulTaskNotifyTake(pdTRUE, portMAX_DELAY);
        uint16_t raw[ADC_NUM_CHANNELS];
        adc_dma_block_average(raw);

        uint32_t lambda_mv = (raw[ADC_IDX_LAMBDA] * ADC_VREF_MV + ADC_FULL_SCALE / 2u) / ADC_FULL_SCALE;
        injector_set(60000u, lambda_mv);

        can_frame_t frame = {.id = 0x100, .len = 6};
        for(uint32_t c = 0; c < ADC_NUM_CHANNELS; c++) {
            frame.data[2 * c] = (uint8_t)(raw[c] & 0xFFu);
            frame.data[2 * c + 1] = (uint8_t)(raw[c] >> 8);
        }
        canbus_send(&frame);

        if(++blocks % 10u == 0u) {
            log_printf("ADC lambda=%u tps=%u rpm=%u pulse=%lu us\r\n",
                       raw[ADC_IDX_LAMBDA], raw[ADC_IDX_TPS], raw[ADC_IDX_RPM],
                       (unsigned long)injector_pulse_us());
        }
    }
}

void app_start(void)
{
    xTaskCreate(control_loop, "control", 256, NULL, 3, &control_task);
    vTaskStartScheduler();
}
