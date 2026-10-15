#include "unity.h"
#include "params.h"
#include "sensors.h"

void setUp(void) {}
void tearDown(void) {}

static uint16_t counts(float volts)
{
    return (uint16_t)(volts / 3.3f * 4095.0f + 0.5f);
}

static void test_lambda_nos_extremos_e_no_meio(void)
{
    TEST_ASSERT_FLOAT_WITHIN(1e-4f, 0.7f, sensor_lambda(0));
    TEST_ASSERT_FLOAT_WITHIN(1e-4f, 1.3f, sensor_lambda(4095));
    TEST_ASSERT_FLOAT_WITHIN(1e-3f, 1.0f, sensor_lambda(counts(1.65f)));
}

static void test_tps_satura_entre_0_e_100(void)
{
    TEST_ASSERT_EQUAL_FLOAT(0.0f, sensor_tps(0));
    TEST_ASSERT_FLOAT_WITHIN(0.1f, 0.0f, sensor_tps(counts(0.3f)));
    TEST_ASSERT_FLOAT_WITHIN(0.1f, 50.0f, sensor_tps(counts(1.65f)));
    TEST_ASSERT_FLOAT_WITHIN(0.1f, 100.0f, sensor_tps(counts(3.0f)));
    TEST_ASSERT_EQUAL_FLOAT(100.0f, sensor_tps(4095));
}

static void test_rpm_do_conversor_fv(void)
{
    TEST_ASSERT_EQUAL_FLOAT(0.0f, sensor_rpm(0));
    TEST_ASSERT_FLOAT_WITHIN(2.0f, 2000.0f, sensor_rpm(counts(0.825f)));
    TEST_ASSERT_FLOAT_WITHIN(0.01f, 8000.0f, sensor_rpm(4095));
}

static void test_ciclo_do_motor(void)
{
    TEST_ASSERT_EQUAL_UINT32(150000u, engine_cycle_us(800.0f));
    TEST_ASSERT_EQUAL_UINT32(60000u, engine_cycle_us(2000.0f));
    TEST_ASSERT_EQUAL_UINT32(20000u, engine_cycle_us(6000.0f));
    TEST_ASSERT_EQUAL_UINT32(T_CICLO_MAX_US, engine_cycle_us(0.0f));
    TEST_ASSERT_EQUAL_UINT32(T_CICLO_MAX_US, engine_cycle_us(50.0f));
}

int main(void)
{
    UNITY_BEGIN();
    RUN_TEST(test_lambda_nos_extremos_e_no_meio);
    RUN_TEST(test_tps_satura_entre_0_e_100);
    RUN_TEST(test_rpm_do_conversor_fv);
    RUN_TEST(test_ciclo_do_motor);
    return UNITY_END();
}
