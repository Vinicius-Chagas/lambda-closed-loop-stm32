#include <string.h>

#include "unity.h"
#include "can_codec.h"

void setUp(void) {}
void tearDown(void) {}

static void test_estado_layout(void)
{
    msg_ecu_estado_t m = {
        .lambda_med = 1.000f,
        .lambda_alvo = 0.950f,
        .ti_us = 4321,
        .correcao = -0.10f,
        .modo = MODO_MALHA_FECHADA,
        .contador = 0x13, /* só os 4 bits de baixo vão no frame */
    };
    uint8_t d[8];
    can_pack_ecu_estado(&m, d);
    const uint8_t esperado[8] = {0xE8, 0x03, 0xB6, 0x03, 0xE1, 0x10, (uint8_t)-40, 0x31};
    TEST_ASSERT_EQUAL_HEX8_ARRAY(esperado, d, 8);
}

static void test_estado_satura(void)
{
    msg_ecu_estado_t m = {.lambda_med = 99.0f, .lambda_alvo = -1.0f, .ti_us = 100000u, .correcao = 1.0f};
    uint8_t d[8];
    can_pack_ecu_estado(&m, d);
    TEST_ASSERT_EQUAL_HEX8(0xFF, d[0]);
    TEST_ASSERT_EQUAL_HEX8(0xFF, d[1]);
    TEST_ASSERT_EQUAL_HEX8(0x00, d[2]);
    TEST_ASSERT_EQUAL_HEX8(0x00, d[3]);
    TEST_ASSERT_EQUAL_HEX8(0xFF, d[4]);
    TEST_ASSERT_EQUAL_HEX8(0xFF, d[5]);
    TEST_ASSERT_EQUAL_INT8(127, (int8_t)d[6]);
}

static void test_sensores_e_tempos(void)
{
    msg_ecu_sensores_t s = {.rpm = 2000.4f, .tps = 12.34f};
    uint8_t d[8];
    can_pack_ecu_sensores(&s, d);
    const uint8_t es[4] = {0xD0, 0x07, 0x7B, 0x00};
    TEST_ASSERT_EQUAL_HEX8_ARRAY(es, d, 4);

    msg_ecu_tempos_t t = {.periodo_us = 10000, .jitter_max_us = 12, .exec_max_us = 300, .deadlines_perdidos = 2};
    can_pack_ecu_tempos(&t, d);
    const uint8_t et[8] = {0x10, 0x27, 0x0C, 0x00, 0x2C, 0x01, 0x02, 0x00};
    TEST_ASSERT_EQUAL_HEX8_ARRAY(et, d, 8);
}

static void test_comando_do_painel(void)
{
    const uint8_t d[4] = {0xB0, 0x04, 0x00, 0x01};
    msg_painel_ecu_t m;
    TEST_ASSERT_TRUE(can_unpack_painel_ecu(d, 4, &m));
    TEST_ASSERT_FLOAT_WITHIN(1e-6f, 1.2f, m.lambda_alvo);
    TEST_ASSERT_EQUAL_UINT8(MODO_MALHA_ABERTA, m.modo);
    TEST_ASSERT_EQUAL_UINT8(PAINEL_FLAG_ZERAR_TEMPOS, m.flags);
    TEST_ASSERT_FALSE(can_unpack_painel_ecu(d, 3, &m));
}

int main(void)
{
    UNITY_BEGIN();
    RUN_TEST(test_estado_layout);
    RUN_TEST(test_estado_satura);
    RUN_TEST(test_sensores_e_tempos);
    RUN_TEST(test_comando_do_painel);
    return UNITY_END();
}
