/*
 * Cálculo do tempo de injeção (docs/equacoes.md, seção 6):
 *   t_i = t_morto + t_base(TPS, RPM) · (1 + c)
 * com c vindo de um PI sobre o erro de λ (malha fechada) ou c = 0 (malha aberta).
 */
#ifndef CONTROLLER_H
#define CONTROLLER_H

#include <stdint.h>

#include "can_codec.h"

typedef struct {
    float integral;
} controller_t;

typedef struct {
    float lambda_med;
    float lambda_alvo;
    float tps;
    float rpm;
    modo_t modo;
} controller_in_t;

typedef struct {
    uint32_t ti_us;      /* largura do pulso */
    uint32_t ciclo_us;   /* período do pulso (um ciclo do motor) */
    float t_base_us;     /* feedforward da tabela */
    float correcao;      /* c, fração (-PI_LIMITE..PI_LIMITE) */
} controller_out_t;

void controller_init(controller_t *c);
void controller_step(controller_t *c, const controller_in_t *in, controller_out_t *out);

#endif /* CONTROLLER_H */
