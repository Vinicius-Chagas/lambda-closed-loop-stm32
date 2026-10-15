/*
 * Tarefas da ECU.
 *
 * ControlTask (prioridade 3): acordada pelo DMA do ADC a cada 10 ms. Faz a média
 *   das amostras e chama ecu_step (control/ecu.c: conversão, tabela + PI),
 *   atualiza o TIM2 e mede os próprios tempos.
 * CanTask (prioridade 2): a cada 20 ms publica estado e sensores, a cada 100 ms
 *   os tempos; trata os comandos do painel (0x200) e escreve o log na USART.
 *
 * Os dados trocados entre as duas ficam em `shared`, protegidos por seção crítica.
 */
#include <string.h>

#include "ecu_hw.h"
#include "app.h"
#include "can_codec.h"
#include "ecu.h"
#include "params.h"
#include "timing_stats.h"

#define CAN_PERIOD_MS        20u
#define TEMPOS_A_CADA        5u    /* 0x110 a cada 5 × 20 ms */
#define LOG_A_CADA           25u   /* log a cada 25 × 20 ms */

typedef struct {
    /* saídas da ControlTask */
    uint16_t adc[ADC_NUM_CHANNELS];
    float lambda_med;
    float tps;
    float rpm;
    uint32_t ti_us;
    float correcao;
    timing_stats_t tempos;
    /* comandos do painel (CanTask → ControlTask) */
    float lambda_alvo;
    modo_t modo;
    bool zerar_tempos;
} ecu_shared_t;

static ecu_shared_t shared;
static ecu_t ecu; /* só a ControlTask mexe */
static TaskHandle_t control_task;

_Static_assert((int)ADC_IDX_LAMBDA == (int)ECU_ADC_LAMBDA && (int)ADC_IDX_TPS == (int)ECU_ADC_TPS &&
               (int)ADC_IDX_RPM == (int)ECU_ADC_RPM && (int)ADC_NUM_CHANNELS == (int)ECU_ADC_N,
               "ordem dos canais do ADC");

static void control_step(uint32_t ativacoes_puladas)
{
    uint32_t inicio = timing_now_us();
    uint32_t liberacao = adc_dma_block_timestamp_us();

    taskENTER_CRITICAL();
    float lambda_alvo = shared.lambda_alvo;
    modo_t modo = shared.modo;
    taskEXIT_CRITICAL();
    ecu_set_alvo(&ecu, lambda_alvo);
    ecu_set_modo(&ecu, modo);

    uint16_t adc[ADC_NUM_CHANNELS];
    adc_dma_block_average(adc);
    ecu_out_t out;
    ecu_step(&ecu, adc, &out);
    injector_set(out.ctrl.ciclo_us, out.ctrl.ti_us);

    taskENTER_CRITICAL();
    memcpy(shared.adc, adc, sizeof(adc));
    shared.lambda_med = out.lambda_med;
    shared.tps = out.tps;
    shared.rpm = out.rpm;
    shared.ti_us = out.ctrl.ti_us;
    shared.correcao = out.ctrl.correcao;
    if(shared.zerar_tempos) {
        shared.zerar_tempos = false;
        timing_stats_reset(&shared.tempos);
    }
    taskEXIT_CRITICAL();

    uint32_t fim = timing_now_us();
    taskENTER_CRITICAL();
    timing_stats_update(&shared.tempos, liberacao, inicio, fim, ativacoes_puladas);
    taskEXIT_CRITICAL();
}

static void control_loop(void *arg)
{
    (void)arg;

    /* Periféricos iniciados aqui, com o escalonador rodando: antes dele o
     * FreeRTOS deixa as interrupções mascaradas e o HAL_GetTick não anda.
     * A CanTask tem prioridade menor, então só roda depois disto. */
    injector_init();
    if(!canbus_init()) {
        ecu_fatal("CAN init");
    }
    adc_dma_init(xTaskGetCurrentTaskHandle());
    log_printf("ECU pronta\r\n");

    for(;;) {
        uint32_t n = ulTaskNotifyTake(pdTRUE, portMAX_DELAY);
        control_step(n > 1u ? n - 1u : 0u);
    }
}

static void handle_command(const can_frame_t *frame)
{
    msg_painel_ecu_t cmd;
    if(frame->id != CAN_ID_PAINEL_ECU || !can_unpack_painel_ecu(frame->data, frame->len, &cmd)) {
        return;
    }
    taskENTER_CRITICAL();
    if(cmd.lambda_alvo >= LAMBDA_ALVO_MIN && cmd.lambda_alvo <= LAMBDA_ALVO_MAX) {
        shared.lambda_alvo = cmd.lambda_alvo;
    }
    if(cmd.modo == MODO_MALHA_ABERTA || cmd.modo == MODO_MALHA_FECHADA) {
        shared.modo = (modo_t)cmd.modo;
    }
    if(cmd.flags & PAINEL_FLAG_ZERAR_TEMPOS) {
        shared.zerar_tempos = true;
    }
    taskEXIT_CRITICAL();
    log_printf("CMD alvo=%u modo=%u flags=%u\r\n",
               (unsigned)(cmd.lambda_alvo * 1000.0f + 0.5f), cmd.modo, cmd.flags);
}

static void can_loop(void *arg)
{
    (void)arg;
    uint32_t ciclo = 0;
    uint8_t contador = 0;
    TickType_t wake = xTaskGetTickCount();

    for(;;) {
        can_frame_t rx;
        while(canbus_receive(&rx, 0)) {
            handle_command(&rx);
        }

        ecu_shared_t s;
        taskENTER_CRITICAL();
        s = shared;
        taskEXIT_CRITICAL();

        msg_ecu_estado_t estado = {
            .lambda_med = s.lambda_med,
            .lambda_alvo = s.lambda_alvo,
            .ti_us = s.ti_us,
            .correcao = s.correcao,
            .modo = s.modo,
            .contador = contador++,
        };
        can_frame_t tx = {.id = CAN_ID_ECU_ESTADO, .len = 8};
        can_pack_ecu_estado(&estado, tx.data);
        canbus_send(&tx);

        msg_ecu_sensores_t sens = {.rpm = s.rpm, .tps = s.tps};
        tx = (can_frame_t){.id = CAN_ID_ECU_SENSORES, .len = 4};
        can_pack_ecu_sensores(&sens, tx.data);
        canbus_send(&tx);

        if(ciclo % TEMPOS_A_CADA == 0u) {
            msg_ecu_tempos_t tempos = {
                .periodo_us = (uint16_t)(s.tempos.periodo_us > 0xFFFFu ? 0xFFFFu : s.tempos.periodo_us),
                .jitter_max_us = (uint16_t)(s.tempos.jitter_max_us > 0xFFFFu ? 0xFFFFu : s.tempos.jitter_max_us),
                .exec_max_us = (uint16_t)(s.tempos.exec_max_us > 0xFFFFu ? 0xFFFFu : s.tempos.exec_max_us),
                .deadlines_perdidos = (uint16_t)(s.tempos.deadlines_perdidos > 0xFFFFu ? 0xFFFFu : s.tempos.deadlines_perdidos),
            };
            tx = (can_frame_t){.id = CAN_ID_ECU_TEMPOS, .len = 8};
            can_pack_ecu_tempos(&tempos, tx.data);
            canbus_send(&tx);
        }

        if(ciclo % LOG_A_CADA == 0u) {
            log_printf("ADC lambda=%u tps=%u rpm=%u ti=%lu us\r\n",
                       s.adc[ADC_IDX_LAMBDA], s.adc[ADC_IDX_TPS], s.adc[ADC_IDX_RPM],
                       (unsigned long)s.ti_us);
            log_printf("TEMPOS periodo=%lu jitter=%lu exec=%lu perdidos=%lu n=%lu\r\n",
                       (unsigned long)s.tempos.periodo_us, (unsigned long)s.tempos.jitter_max_us,
                       (unsigned long)s.tempos.exec_max_us, (unsigned long)s.tempos.deadlines_perdidos,
                       (unsigned long)s.tempos.ativacoes);
        }
        ciclo++;
        xTaskDelayUntil(&wake, pdMS_TO_TICKS(CAN_PERIOD_MS));
    }
}

void app_start(void)
{
    ecu_init(&ecu);
    shared.lambda_alvo = ecu.lambda_alvo;
    shared.modo = ecu.modo;
    timing_stats_init(&shared.tempos, CONTROL_PERIOD_US);

    if(xTaskCreate(control_loop, "control", 256, NULL, 3, &control_task) != pdPASS ||
       xTaskCreate(can_loop, "can", 320, NULL, 2, NULL) != pdPASS) {
        ecu_fatal("tasks");
    }
    vTaskStartScheduler();
}
