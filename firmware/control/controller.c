#include "controller.h"
#include "fuel_map.h"
#include "params.h"
#include "sensors.h"

void controller_init(controller_t *c)
{
    c->integral = 0.0f;
}

static float minf(float a, float b)
{
    return a < b ? a : b;
}

static float maxf(float a, float b)
{
    return a > b ? a : b;
}

static float clampf(float x, float lo, float hi)
{
    return minf(maxf(x, lo), hi);
}

/* PI com anti-windup condicional: quando a saída satura no sentido em que o
 * erro empurra, a integral só anda até o ponto em que a saída encosta no
 * limite (nunca além), e fica sempre dentro de ±PI_LIMITE. */
static float pi_step(controller_t *c, float erro)
{
    float p = PI_KP * erro;
    float i = c->integral + PI_KI * erro * CONTROL_DT_S;
    if(erro > 0.0f && p + i > PI_LIMITE) {
        i = maxf(c->integral, PI_LIMITE - p);
    } else if(erro < 0.0f && p + i < -PI_LIMITE) {
        i = minf(c->integral, -PI_LIMITE - p);
    }
    c->integral = clampf(i, -PI_LIMITE, PI_LIMITE);
    return clampf(p + c->integral, -PI_LIMITE, PI_LIMITE);
}

void controller_step(controller_t *c, const controller_in_t *in, controller_out_t *out)
{
    out->ciclo_us = engine_cycle_us(in->rpm);
    out->t_base_us = fuel_map_base_us(in->tps, in->rpm);

    if(in->rpm < RPM_MIN_GIRANDO) { /* motor parado: não injeta */
        controller_init(c);
        out->correcao = 0.0f;
        out->ti_us = 0;
        return;
    }

    if(in->modo == MODO_MALHA_FECHADA) {
        /* e > 0: mistura pobre → mais combustível */
        out->correcao = pi_step(c, in->lambda_med - in->lambda_alvo);
    } else {
        controller_init(c);
        out->correcao = 0.0f;
    }

    float ti = T_MORTO_US + out->t_base_us * (1.0f + out->correcao);
    float ti_max = (float)out->ciclo_us * 0.9f; /* bico nunca 100 % aberto */
    if(ti_max > (float)TI_MAX_US) {
        ti_max = (float)TI_MAX_US;
    }
    out->ti_us = (uint32_t)(clampf(ti, 0.0f, ti_max) + 0.5f);
}
