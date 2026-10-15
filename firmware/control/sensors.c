#include "sensors.h"
#include "params.h"

static float counts_to_volts(uint16_t counts)
{
    return (float)counts * (ADC_VREF_V / ADC_COUNTS);
}

static float clampf(float x, float lo, float hi)
{
    return x < lo ? lo : (x > hi ? hi : x);
}

float sensor_lambda(uint16_t counts)
{
    return LAMBDA_V_MIN + LAMBDA_V_SPAN * counts_to_volts(counts) / ADC_VREF_V;
}

float sensor_tps(uint16_t counts)
{
    float tps = (counts_to_volts(counts) - TPS_V_MIN) / TPS_V_SPAN * 100.0f;
    return clampf(tps, 0.0f, 100.0f);
}

float sensor_rpm(uint16_t counts)
{
    return counts_to_volts(counts) / ADC_VREF_V * RPM_FULL_SCALE;
}

uint32_t engine_cycle_us(float rpm)
{
    if(rpm < RPM_MIN_GIRANDO) {
        return T_CICLO_MAX_US;
    }
    return (uint32_t)(120.0e6f / rpm + 0.5f);
}
