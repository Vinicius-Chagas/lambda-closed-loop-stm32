/*
 * Constantes de hardware da ECU (STM32F072).
 *
 * Mapa de pinos:
 *   PA0  ADC_IN0  sonda lambda (banda larga, 0..3,3 V)
 *   PA1  ADC_IN1  TPS - posição da borboleta
 *   PA2  ADC_IN2  rotação (saída de um conversor F/V tipo LM2907)
 *   PA5  TIM2_CH1 pulso do bico injetor
 *   PA9  USART1_TX log de depuração (115200 8N1)
 *   PA10 USART1_RX
 *   PB8  CAN_RX
 *   PB9  CAN_TX
 */
#ifndef BOARD_H
#define BOARD_H

#include <stdint.h>

#define SYSCLK_HZ            48000000u
#define TIMER_TICK_HZ        1000000u   /* timers de medição/injeção contam em µs */
#define TIMER_PRESCALER      ((SYSCLK_HZ / TIMER_TICK_HZ) - 1u)

#define ADC_NUM_CHANNELS     3u
#define ADC_SAMPLE_RATE_HZ   1000u      /* disparo por TIM15 TRGO */
#define ADC_SETS_PER_BLOCK   10u        /* 10 varreduras = 10 ms por bloco */
#define ADC_VREF_MV          3300u
#define ADC_FULL_SCALE       4095u

#define CONTROL_PERIOD_US    10000u     /* período nominal da ControlTask */

enum adc_index {
    ADC_IDX_LAMBDA = 0,
    ADC_IDX_TPS    = 1,
    ADC_IDX_RPM    = 2,
};

#define CAN_BITRATE          500000u

#endif /* BOARD_H */
