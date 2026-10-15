#include "unity.h"
#include "timing_stats.h"

static timing_stats_t s;

void setUp(void)
{
    timing_stats_init(&s, 10000u);
}
void tearDown(void) {}

static void test_periodo_jitter_e_execucao(void)
{
    timing_stats_update(&s, 0, 20, 120, 0);
    TEST_ASSERT_EQUAL_UINT32(0, s.jitter_max_us); /* sem período ainda */
    timing_stats_update(&s, 10000, 10050, 10350, 0);
    TEST_ASSERT_EQUAL_UINT32(10030, s.periodo_us);
    TEST_ASSERT_EQUAL_UINT32(30, s.jitter_max_us);
    timing_stats_update(&s, 20000, 20000, 20100, 0);
    TEST_ASSERT_EQUAL_UINT32(9950, s.periodo_us);
    TEST_ASSERT_EQUAL_UINT32(50, s.jitter_max_us);
    TEST_ASSERT_EQUAL_UINT32(100, s.exec_us);
    TEST_ASSERT_EQUAL_UINT32(300, s.exec_max_us);
    TEST_ASSERT_EQUAL_UINT32(0, s.deadlines_perdidos);
    TEST_ASSERT_EQUAL_UINT32(3, s.ativacoes);
}

static void test_deadline_perdido_por_atraso_e_por_ativacao_pulada(void)
{
    timing_stats_update(&s, 0, 9000, 10001, 0); /* terminou depois da próxima liberação */
    TEST_ASSERT_EQUAL_UINT32(1, s.deadlines_perdidos);
    timing_stats_update(&s, 30000, 30010, 30100, 1); /* uma liberação sem execução */
    TEST_ASSERT_EQUAL_UINT32(2, s.deadlines_perdidos);
    /* com ativação pulada, o intervalo não conta como período */
    TEST_ASSERT_EQUAL_UINT32(0, s.jitter_max_us);
}

static void test_contador_que_estoura(void)
{
    timing_stats_update(&s, 0xFFFFF000u, 0xFFFFF000u, 0xFFFFF010u, 0);
    timing_stats_update(&s, 0x00001710u, 0x00001710u, 0x00001720u, 0);
    TEST_ASSERT_EQUAL_UINT32(10000, s.periodo_us);
    TEST_ASSERT_EQUAL_UINT32(0, s.jitter_max_us);
    TEST_ASSERT_EQUAL_UINT32(0, s.deadlines_perdidos);
}

static void test_reset_mantem_periodo_nominal(void)
{
    timing_stats_update(&s, 0, 0, 5000, 0);
    timing_stats_reset(&s);
    TEST_ASSERT_EQUAL_UINT32(0, s.exec_max_us);
    TEST_ASSERT_EQUAL_UINT32(10000, s.periodo_nominal_us);
    TEST_ASSERT_FALSE(s.tem_inicio);
}

int main(void)
{
    UNITY_BEGIN();
    RUN_TEST(test_periodo_jitter_e_execucao);
    RUN_TEST(test_deadline_perdido_por_atraso_e_por_ativacao_pulada);
    RUN_TEST(test_contador_que_estoura);
    RUN_TEST(test_reset_mantem_periodo_nominal);
    return UNITY_END();
}
