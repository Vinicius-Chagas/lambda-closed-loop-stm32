/*
 * Lógica da ECU a cada período de controle, sem dependência de hardware:
 * contagens do ADC → grandezas → tempo de injeção. Usada pela ControlTask do
 * firmware e pelo teste SIL (go/internal/sil), que compila este mesmo código.
 */
#ifndef ECU_H
#define ECU_H

#include <stdint.h>

#include "controller.h"

enum { ECU_ADC_LAMBDA = 0, ECU_ADC_TPS = 1, ECU_ADC_RPM = 2, ECU_ADC_N = 3 };

typedef struct {
    controller_t ctrl;
    float lambda_alvo;
    modo_t modo;
} ecu_t;

typedef struct {
    float lambda_med;
    float tps;
    float rpm;
    controller_out_t ctrl;
} ecu_out_t;

void ecu_init(ecu_t *e);
void ecu_set_alvo(ecu_t *e, float lambda_alvo);   /* ignora fora de LAMBDA_ALVO_MIN..MAX */
void ecu_set_modo(ecu_t *e, modo_t modo);
void ecu_step(ecu_t *e, const uint16_t adc[ECU_ADC_N], ecu_out_t *out);

#endif /* ECU_H */
