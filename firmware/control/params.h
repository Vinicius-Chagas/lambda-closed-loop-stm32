/*
 * Parâmetros do modelo e do controle (ver docs/equacoes.md).
 * Mantenha em sincronia com go/internal/engine/params.go.
 */
#ifndef PARAMS_H
#define PARAMS_H

/* Combustível e motor */
#define AFR_ST              14.7f    /* relação ar/combustível estequiométrica */
#define M_AR_MAX_MG         480.0f   /* massa de ar máxima por cilindro, por ciclo */
#define RPM_LENTA           800.0f
#define RPM_MAX             6000.0f

/* Bico injetor */
#define Q_INJ_MG_PER_MS     3.0f     /* vazão */
#define T_MORTO_US          800.0f   /* tempo morto (abertura sem vazão) */

/* Sensores (entrada do ADC, 0..3,3 V) */
#define ADC_VREF_V          3.3f
#define ADC_COUNTS          4095.0f
#define LAMBDA_V_MIN        0.7f     /* λ que dá 0 V */
#define LAMBDA_V_SPAN       0.6f     /* faixa de λ em 0..3,3 V */
#define TPS_V_MIN           0.3f     /* tensão com borboleta fechada */
#define TPS_V_SPAN          2.7f     /* 0,3..3,0 V = 0..100 % */
#define RPM_FULL_SCALE      8000.0f  /* rpm que dá 3,3 V no conversor F/V */

/* Controle */
#define CONTROL_DT_S        0.010f
#define LAMBDA_ALVO_PADRAO  1.0f
#define LAMBDA_ALVO_MIN     0.80f
#define LAMBDA_ALVO_MAX     1.20f
#define PI_KP               0.15f
#define PI_KI               2.0f     /* 1/s */
#define PI_LIMITE           0.30f    /* correção máxima: ±30 % (ver equacoes.md) */

/* Injetor: um pulso por ciclo (2 voltas); abaixo disso o motor está parado */
#define RPM_MIN_GIRANDO     100.0f
#define T_CICLO_MAX_US      1200000u /* 120 / 100 rpm */
#define TI_MAX_US           60000u

#endif /* PARAMS_H */
