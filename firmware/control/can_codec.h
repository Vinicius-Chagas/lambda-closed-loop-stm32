/*
 * Frames CAN da bancada (IDs de 11 bits, dados little-endian).
 * Mesma definição de can/lambda.dbc e de go/internal/canbus.
 */
#ifndef CAN_CODEC_H
#define CAN_CODEC_H

#include <stdbool.h>
#include <stdint.h>

#define CAN_ID_ECU_ESTADO     0x100u
#define CAN_ID_ECU_SENSORES   0x101u
#define CAN_ID_ECU_TEMPOS     0x110u
#define CAN_ID_PAINEL_ECU     0x200u
#define CAN_ID_PLANTA         0x300u
#define CAN_ID_PAINEL_PLANTA  0x310u

typedef enum { MODO_MALHA_ABERTA = 0, MODO_MALHA_FECHADA = 1 } modo_t;

/* 0x100, 8 bytes */
typedef struct {
    float lambda_med;      /* u16, 0,001 */
    float lambda_alvo;     /* u16, 0,001 */
    uint32_t ti_us;        /* u16, µs (satura em 65535) */
    float correcao;        /* i8, 0,25 % (-0,30..0,30 → -120..120) */
    modo_t modo;           /* byte 7, bit 0 */
    uint8_t contador;      /* byte 7, bits 4..7: incrementa a cada frame (0..15) */
} msg_ecu_estado_t;

/* 0x101, 4 bytes */
typedef struct {
    float rpm;             /* u16, 1 rpm */
    float tps;             /* u16, 0,1 % */
} msg_ecu_sensores_t;

/* 0x110, 8 bytes; tempos da ControlTask desde a última reinicialização */
typedef struct {
    uint16_t periodo_us;          /* último período medido */
    uint16_t jitter_max_us;
    uint16_t exec_max_us;
    uint16_t deadlines_perdidos;
} msg_ecu_tempos_t;

/* 0x200, 4 bytes */
#define PAINEL_MANTER_MODO   0xFFu
#define PAINEL_FLAG_ZERAR_TEMPOS 0x01u
typedef struct {
    float lambda_alvo;     /* u16, 0,001; 0 = mantém */
    uint8_t modo;          /* u8: 0 aberta, 1 fechada, 0xFF mantém */
    uint8_t flags;         /* bit0: zera as estatísticas de tempo */
} msg_painel_ecu_t;

void can_pack_ecu_estado(const msg_ecu_estado_t *m, uint8_t data[8]);
void can_pack_ecu_sensores(const msg_ecu_sensores_t *m, uint8_t data[4]);
void can_pack_ecu_tempos(const msg_ecu_tempos_t *m, uint8_t data[8]);
bool can_unpack_painel_ecu(const uint8_t *data, uint8_t len, msg_painel_ecu_t *m);

#endif /* CAN_CODEC_H */
