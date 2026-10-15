/*
 * Camada de hardware da ECU: relógio, base de tempo em µs, ADC+DMA,
 * bico injetor, CAN e log serial. Toda a dependência do HAL fica aqui;
 * a lógica de controle (firmware/control) não conhece o HAL.
 */
#ifndef ECU_HW_H
#define ECU_HW_H

#include <stdbool.h>
#include <stdint.h>

#include "board.h"
#include "FreeRTOS.h"
#include "task.h"

/* clock.c ---------------------------------------------------------------- */
/* Configura SYSCLK = 48 MHz (HSE 8 MHz x6, com HSI48 como reserva). */
void clock_init(void);

/* timing.c --------------------------------------------------------------- */
/* TIM7 contando em µs, estendido para 32 bits pela interrupção de estouro. */
void timing_init(void);
uint32_t timing_now_us(void);

/* adc_dma.c -------------------------------------------------------------- */
/* Varredura IN0..IN2 disparada pelo TIM15 a 1 kHz, DMA circular com dois
 * blocos de 10 varreduras. A cada bloco completo a tarefa `notify` recebe
 * uma notificação (a cada 10 ms). */
void adc_dma_init(TaskHandle_t notify);
/* Copia a média (em contagens de 12 bits) do último bloco completo. */
void adc_dma_block_average(uint16_t out[ADC_NUM_CHANNELS]);
/* Instante (µs) em que o último bloco ficou pronto. */
uint32_t adc_dma_block_timestamp_us(void);

/* injector.c ------------------------------------------------------------- */
/* TIM2 (32 bits, 1 µs): um pulso por ciclo do motor. */
void injector_init(void);
void injector_set(uint32_t cycle_period_us, uint32_t pulse_us);
uint32_t injector_pulse_us(void);

/* canbus.c --------------------------------------------------------------- */
typedef struct {
    uint32_t id;
    uint8_t len;
    uint8_t data[8];
} can_frame_t;

bool canbus_init(void);
bool canbus_send(const can_frame_t *frame);
/* Retira um frame recebido da fila (espera até `timeout` ticks). */
bool canbus_receive(can_frame_t *frame, TickType_t timeout);

/* log.c ------------------------------------------------------------------ */
void log_init(void);
void log_printf(const char *fmt, ...) __attribute__((format(printf, 1, 2)));

/* Falha fatal: registra no log e para. */
void ecu_fatal(const char *what);

#endif /* ECU_HW_H */
