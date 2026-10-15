#include "can_codec.h"

static void put_u16(uint8_t *p, uint32_t v)
{
    if(v > 0xFFFFu) {
        v = 0xFFFFu;
    }
    p[0] = (uint8_t)(v & 0xFFu);
    p[1] = (uint8_t)(v >> 8);
}

static uint16_t get_u16(const uint8_t *p)
{
    return (uint16_t)(p[0] | (p[1] << 8));
}

static uint32_t scale_u16(float x, float res)
{
    if(x <= 0.0f) {
        return 0u;
    }
    float v = x / res + 0.5f;
    return v >= 65535.0f ? 0xFFFFu : (uint32_t)v;
}

static int8_t scale_i8(float x, float res)
{
    float v = x / res;
    v += v >= 0.0f ? 0.5f : -0.5f;
    if(v > 127.0f) {
        return 127;
    }
    if(v < -127.0f) {
        return -127;
    }
    return (int8_t)v;
}

void can_pack_ecu_estado(const msg_ecu_estado_t *m, uint8_t data[8])
{
    put_u16(&data[0], scale_u16(m->lambda_med, 0.001f));
    put_u16(&data[2], scale_u16(m->lambda_alvo, 0.001f));
    put_u16(&data[4], m->ti_us);
    data[6] = (uint8_t)scale_i8(m->correcao, 0.0025f);
    data[7] = (uint8_t)((m->modo & 0x01u) | (uint8_t)(m->contador << 4));
}

void can_pack_ecu_sensores(const msg_ecu_sensores_t *m, uint8_t data[4])
{
    put_u16(&data[0], scale_u16(m->rpm, 1.0f));
    put_u16(&data[2], scale_u16(m->tps, 0.1f));
}

void can_pack_ecu_tempos(const msg_ecu_tempos_t *m, uint8_t data[8])
{
    put_u16(&data[0], m->periodo_us);
    put_u16(&data[2], m->jitter_max_us);
    put_u16(&data[4], m->exec_max_us);
    put_u16(&data[6], m->deadlines_perdidos);
}

bool can_unpack_painel_ecu(const uint8_t *data, uint8_t len, msg_painel_ecu_t *m)
{
    if(len < 4u) {
        return false;
    }
    m->lambda_alvo = (float)get_u16(&data[0]) * 0.001f;
    m->modo = data[2];
    m->flags = data[3];
    return true;
}
