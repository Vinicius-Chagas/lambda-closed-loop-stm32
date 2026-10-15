/*
 * Estatísticas de tempo real de uma tarefa periódica.
 *
 * - período: intervalo entre dois inícios consecutivos;
 * - jitter: |período − período nominal|, máximo desde a última reinicialização;
 * - execução: fim − início, máximo;
 * - deadline perdido: a tarefa terminou depois da próxima liberação
 *   (fim − liberação > período nominal) ou uma ativação foi pulada.
 *
 * Tempos em µs de um contador livre de 32 bits (a subtração trata o estouro).
 */
#ifndef TIMING_STATS_H
#define TIMING_STATS_H

#include <stdbool.h>
#include <stdint.h>

typedef struct {
    uint32_t periodo_nominal_us;
    uint32_t ultimo_inicio_us;
    bool tem_inicio;
    uint32_t periodo_us;
    uint32_t jitter_max_us;
    uint32_t exec_us;
    uint32_t exec_max_us;
    uint32_t deadlines_perdidos;
    uint32_t ativacoes;
} timing_stats_t;

void timing_stats_init(timing_stats_t *s, uint32_t periodo_nominal_us);

/* Zera máximos e contadores (mantém o período nominal). */
void timing_stats_reset(timing_stats_t *s);

/* Uma execução: liberação (evento que acordou a tarefa), início e fim.
 * ativacoes_puladas: liberações que aconteceram sem a tarefa rodar. */
void timing_stats_update(timing_stats_t *s, uint32_t liberacao_us, uint32_t inicio_us,
                         uint32_t fim_us, uint32_t ativacoes_puladas);

#endif /* TIMING_STATS_H */
