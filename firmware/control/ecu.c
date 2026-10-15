#include "ecu.h"
#include "params.h"
#include "sensors.h"

void ecu_init(ecu_t *e)
{
    controller_init(&e->ctrl);
    e->lambda_alvo = LAMBDA_ALVO_PADRAO;
    e->modo = MODO_MALHA_FECHADA;
}

void ecu_set_alvo(ecu_t *e, float lambda_alvo)
{
    if(lambda_alvo >= LAMBDA_ALVO_MIN && lambda_alvo <= LAMBDA_ALVO_MAX) {
        e->lambda_alvo = lambda_alvo;
    }
}

void ecu_set_modo(ecu_t *e, modo_t modo)
{
    if(modo == MODO_MALHA_ABERTA || modo == MODO_MALHA_FECHADA) {
        e->modo = modo;
    }
}

void ecu_step(ecu_t *e, const uint16_t adc[ECU_ADC_N], ecu_out_t *out)
{
    out->lambda_med = sensor_lambda(adc[ECU_ADC_LAMBDA]);
    out->tps = sensor_tps(adc[ECU_ADC_TPS]);
    out->rpm = sensor_rpm(adc[ECU_ADC_RPM]);

    controller_in_t in = {
        .lambda_med = out->lambda_med,
        .lambda_alvo = e->lambda_alvo,
        .tps = out->tps,
        .rpm = out->rpm,
        .modo = e->modo,
    };
    controller_step(&e->ctrl, &in, &out->ctrl);
}
