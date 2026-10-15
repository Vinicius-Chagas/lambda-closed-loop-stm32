/*
 * Conversão das contagens do ADC (12 bits) em grandezas físicas.
 * Inverso das equações dos sensores em docs/equacoes.md, seção 5.
 */
#ifndef SENSORS_H
#define SENSORS_H

#include <stdint.h>

float sensor_lambda(uint16_t counts);  /* λ, 0,7..1,3 */
float sensor_tps(uint16_t counts);     /* abertura da borboleta, 0..100 % */
float sensor_rpm(uint16_t counts);     /* rotação, 0..8000 rpm */

/* Duração de um ciclo do motor (2 voltas) em µs; motor parado → T_CICLO_MAX_US. */
uint32_t engine_cycle_us(float rpm);

#endif /* SENSORS_H */
