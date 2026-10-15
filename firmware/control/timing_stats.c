#include "timing_stats.h"

void timing_stats_init(timing_stats_t *s, uint32_t periodo_nominal_us)
{
    s->periodo_nominal_us = periodo_nominal_us;
    timing_stats_reset(s);
}

void timing_stats_reset(timing_stats_t *s)
{
    s->tem_inicio = false;
    s->ultimo_inicio_us = 0;
    s->periodo_us = 0;
    s->jitter_max_us = 0;
    s->exec_us = 0;
    s->exec_max_us = 0;
    s->deadlines_perdidos = 0;
    s->ativacoes = 0;
}

void timing_stats_update(timing_stats_t *s, uint32_t liberacao_us, uint32_t inicio_us,
                         uint32_t fim_us, uint32_t ativacoes_puladas)
{
    if(s->tem_inicio && ativacoes_puladas == 0u) {
        s->periodo_us = inicio_us - s->ultimo_inicio_us;
        uint32_t jitter = s->periodo_us > s->periodo_nominal_us
                              ? s->periodo_us - s->periodo_nominal_us
                              : s->periodo_nominal_us - s->periodo_us;
        if(jitter > s->jitter_max_us) {
            s->jitter_max_us = jitter;
        }
    }
    s->ultimo_inicio_us = inicio_us;
    s->tem_inicio = true;

    s->exec_us = fim_us - inicio_us;
    if(s->exec_us > s->exec_max_us) {
        s->exec_max_us = s->exec_us;
    }

    s->deadlines_perdidos += ativacoes_puladas;
    if(fim_us - liberacao_us > s->periodo_nominal_us) {
        s->deadlines_perdidos++;
    }
    s->ativacoes++;
}
