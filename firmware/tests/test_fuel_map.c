#include <math.h>

#include "unity.h"
#include "fuel_map.h"
#include "params.h"

void setUp(void) {}
void tearDown(void) {}

/* Modelo nominal de docs/equacoes.md: m_ar / (AFR_st · Q_inj), em µs. */
static float modelo_us(float tps, float rpm)
{
    float f = 0.10f + 0.90f * tps / 100.0f;
    float x = (rpm - 4000.0f) / 3000.0f;
    float eta = 0.85f - 0.10f * x * x;
    return M_AR_MAX_MG * f * eta / (AFR_ST * Q_INJ_MG_PER_MS) * 1000.0f;
}

static void test_pontos_da_tabela_seguem_o_modelo(void)
{
    for(int i = 0; i < FUEL_MAP_N_TPS; i++) {
        for(int j = 0; j < FUEL_MAP_N_RPM; j++) {
            float tps = fuel_map_tps[i], rpm = fuel_map_rpm[j];
            TEST_ASSERT_FLOAT_WITHIN(1.0f, modelo_us(tps, rpm), fuel_map_us[i][j]);
            TEST_ASSERT_FLOAT_WITHIN(1e-3f, fuel_map_us[i][j], fuel_map_base_us(tps, rpm));
        }
    }
}

static void test_interpolacao_bilinear(void)
{
    /* meio da célula = média dos 4 cantos */
    float esperado = (fuel_map_us[2][3] + fuel_map_us[2][4] + fuel_map_us[3][3] + fuel_map_us[3][4]) / 4.0f;
    TEST_ASSERT_FLOAT_WITHIN(1e-2f, esperado, fuel_map_base_us(30.0f, 3000.0f));
    /* ponto de operação do teste SIL (2000 rpm): erro de interpolação < 1 % do modelo */
    float tps = 23.08f;
    TEST_ASSERT_FLOAT_WITHIN(0.01f * modelo_us(tps, 2000.0f), modelo_us(tps, 2000.0f),
                             fuel_map_base_us(tps, 2000.0f));
}

static void test_fora_da_tabela_usa_a_borda(void)
{
    TEST_ASSERT_EQUAL_FLOAT(fuel_map_us[0][0], fuel_map_base_us(-5.0f, 0.0f));
    TEST_ASSERT_EQUAL_FLOAT(fuel_map_us[FUEL_MAP_N_TPS - 1][FUEL_MAP_N_RPM - 1],
                            fuel_map_base_us(120.0f, 12000.0f));
    TEST_ASSERT_EQUAL_FLOAT(fuel_map_base_us(50.0f, 8000.0f), fuel_map_base_us(50.0f, 9000.0f));
}

static void test_cresce_com_a_borboleta(void)
{
    for(float rpm = 500.0f; rpm <= 8000.0f; rpm += 250.0f) {
        float anterior = 0.0f;
        for(float tps = 0.0f; tps <= 100.0f; tps += 5.0f) {
            float t = fuel_map_base_us(tps, rpm);
            TEST_ASSERT_TRUE(t > anterior);
            anterior = t;
        }
    }
}

int main(void)
{
    UNITY_BEGIN();
    RUN_TEST(test_pontos_da_tabela_seguem_o_modelo);
    RUN_TEST(test_interpolacao_bilinear);
    RUN_TEST(test_fora_da_tabela_usa_a_borda);
    RUN_TEST(test_cresce_com_a_borboleta);
    return UNITY_END();
}
