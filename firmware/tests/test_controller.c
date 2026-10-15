#include "unity.h"
#include "controller.h"
#include "ecu.h"
#include "fuel_map.h"
#include "params.h"

void setUp(void) {}
void tearDown(void) {}

static controller_in_t entrada(float lambda_med, modo_t modo)
{
    return (controller_in_t){
        .lambda_med = lambda_med, .lambda_alvo = 1.0f, .tps = 23.0f, .rpm = 2000.0f, .modo = modo,
    };
}

static controller_out_t passos(controller_t *c, controller_in_t in, int n)
{
    controller_out_t out = {0};
    for(int k = 0; k < n; k++) {
        controller_step(c, &in, &out);
    }
    return out;
}

static void test_malha_aberta_usa_so_a_tabela(void)
{
    controller_t c;
    controller_init(&c);
    controller_out_t out = passos(&c, entrada(1.1f, MODO_MALHA_ABERTA), 50);
    float base = fuel_map_base_us(23.0f, 2000.0f);
    TEST_ASSERT_EQUAL_FLOAT(0.0f, out.correcao);
    TEST_ASSERT_FLOAT_WITHIN(0.6f, T_MORTO_US + base, (float)out.ti_us);
    TEST_ASSERT_EQUAL_FLOAT(base, out.t_base_us);
    TEST_ASSERT_EQUAL_UINT32(60000u, out.ciclo_us);
}

static void test_mistura_pobre_aumenta_o_combustivel(void)
{
    controller_t c;
    controller_init(&c);
    controller_out_t a = passos(&c, entrada(1.05f, MODO_MALHA_FECHADA), 1);
    controller_out_t b = passos(&c, entrada(1.05f, MODO_MALHA_FECHADA), 10);
    TEST_ASSERT_TRUE(a.correcao > 0.0f);
    TEST_ASSERT_TRUE(b.correcao > a.correcao); /* a parte integral acumula */
    TEST_ASSERT_TRUE(b.ti_us > a.ti_us);
    /* 1º passo: Kp·e + Ki·e·dt */
    TEST_ASSERT_FLOAT_WITHIN(1e-5f, 0.05f * (PI_KP + PI_KI * CONTROL_DT_S), a.correcao);
}

static void test_mistura_rica_reduz_o_combustivel(void)
{
    controller_t c;
    controller_init(&c);
    controller_out_t out = passos(&c, entrada(0.95f, MODO_MALHA_FECHADA), 10);
    TEST_ASSERT_TRUE(out.correcao < 0.0f);
    TEST_ASSERT_TRUE((float)out.ti_us < T_MORTO_US + out.t_base_us);
}

static void test_erro_zero_mantem_a_correcao(void)
{
    controller_t c;
    controller_init(&c);
    controller_out_t a = passos(&c, entrada(1.05f, MODO_MALHA_FECHADA), 20);
    controller_out_t b = passos(&c, entrada(1.0f, MODO_MALHA_FECHADA), 20);
    /* sem erro sobra só a integral: Ki·e·dt·20 */
    TEST_ASSERT_FLOAT_WITHIN(1e-4f, PI_KI * 0.05f * CONTROL_DT_S * 20.0f, b.correcao);
    TEST_ASSERT_TRUE(b.correcao < a.correcao);
}

static void test_correcao_satura_e_nao_acumula(void)
{
    controller_t c;
    controller_init(&c);
    controller_out_t out = passos(&c, entrada(1.3f, MODO_MALHA_FECHADA), 500); /* 5 s pobre */
    TEST_ASSERT_EQUAL_FLOAT(PI_LIMITE, out.correcao);
    TEST_ASSERT_TRUE(c.integral <= PI_LIMITE);

    /* Com anti-windup, ao inverter o erro a correção sai do limite já no 1º passo. */
    out = passos(&c, entrada(0.95f, MODO_MALHA_FECHADA), 1);
    TEST_ASSERT_TRUE(out.correcao < PI_LIMITE);

    controller_init(&c);
    out = passos(&c, entrada(0.7f, MODO_MALHA_FECHADA), 500);
    TEST_ASSERT_EQUAL_FLOAT(-PI_LIMITE, out.correcao);
    TEST_ASSERT_TRUE(c.integral >= -PI_LIMITE);
}

static void test_malha_aberta_zera_a_integral(void)
{
    controller_t c;
    controller_init(&c);
    passos(&c, entrada(1.1f, MODO_MALHA_FECHADA), 30);
    TEST_ASSERT_TRUE(c.integral > 0.0f);
    passos(&c, entrada(1.1f, MODO_MALHA_ABERTA), 1);
    TEST_ASSERT_EQUAL_FLOAT(0.0f, c.integral);
}

static void test_motor_parado_nao_injeta(void)
{
    controller_t c;
    controller_init(&c);
    passos(&c, entrada(1.1f, MODO_MALHA_FECHADA), 30);
    controller_in_t in = entrada(1.1f, MODO_MALHA_FECHADA);
    in.rpm = 50.0f;
    controller_out_t out = passos(&c, in, 1);
    TEST_ASSERT_EQUAL_UINT32(0u, out.ti_us);
    TEST_ASSERT_EQUAL_UINT32(T_CICLO_MAX_US, out.ciclo_us);
    TEST_ASSERT_EQUAL_FLOAT(0.0f, c.integral);
}

static void test_pulso_limitado_a_90_por_cento_do_ciclo(void)
{
    controller_t c;
    controller_init(&c);
    controller_in_t in = {.lambda_med = 1.0f, .lambda_alvo = 1.0f, .tps = 100.0f, .rpm = 20000.0f,
                          .modo = MODO_MALHA_ABERTA};
    controller_out_t out = passos(&c, in, 1);
    TEST_ASSERT_EQUAL_UINT32(6000u, out.ciclo_us);
    TEST_ASSERT_EQUAL_UINT32(5400u, out.ti_us);
}

static uint16_t counts(float volts)
{
    return (uint16_t)(volts / 3.3f * 4095.0f + 0.5f);
}

static void test_ecu_converte_e_aceita_comandos_validos(void)
{
    ecu_t e;
    ecu_init(&e);
    TEST_ASSERT_EQUAL_FLOAT(LAMBDA_ALVO_PADRAO, e.lambda_alvo);
    TEST_ASSERT_EQUAL(MODO_MALHA_FECHADA, e.modo);

    ecu_set_alvo(&e, 1.5f);
    TEST_ASSERT_EQUAL_FLOAT(LAMBDA_ALVO_PADRAO, e.lambda_alvo);
    ecu_set_alvo(&e, 0.9f);
    TEST_ASSERT_EQUAL_FLOAT(0.9f, e.lambda_alvo);
    ecu_set_modo(&e, (modo_t)7);
    TEST_ASSERT_EQUAL(MODO_MALHA_FECHADA, e.modo);
    ecu_set_modo(&e, MODO_MALHA_ABERTA);
    TEST_ASSERT_EQUAL(MODO_MALHA_ABERTA, e.modo);

    /* 1,65 V → λ 1,0; 0,5 V → 7,4 % TPS; 3,0 V → 7273 rpm */
    const uint16_t adc[ECU_ADC_N] = {counts(1.65f), counts(0.5f), counts(3.0f)};
    ecu_out_t out;
    ecu_step(&e, adc, &out);
    TEST_ASSERT_FLOAT_WITHIN(1e-3f, 1.0f, out.lambda_med);
    TEST_ASSERT_FLOAT_WITHIN(0.1f, 7.4f, out.tps);
    TEST_ASSERT_FLOAT_WITHIN(1.0f, 7273.0f, out.rpm);
    TEST_ASSERT_EQUAL_UINT32((uint32_t)(T_MORTO_US + fuel_map_base_us(out.tps, out.rpm) + 0.5f),
                             out.ctrl.ti_us);
}

int main(void)
{
    UNITY_BEGIN();
    RUN_TEST(test_malha_aberta_usa_so_a_tabela);
    RUN_TEST(test_mistura_pobre_aumenta_o_combustivel);
    RUN_TEST(test_mistura_rica_reduz_o_combustivel);
    RUN_TEST(test_erro_zero_mantem_a_correcao);
    RUN_TEST(test_correcao_satura_e_nao_acumula);
    RUN_TEST(test_malha_aberta_zera_a_integral);
    RUN_TEST(test_motor_parado_nao_injeta);
    RUN_TEST(test_pulso_limitado_a_90_por_cento_do_ciclo);
    RUN_TEST(test_ecu_converte_e_aceita_comandos_validos);
    return UNITY_END();
}
