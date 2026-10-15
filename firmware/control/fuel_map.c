#include "fuel_map.h"

const float fuel_map_tps[FUEL_MAP_N_TPS] = {0, 10, 20, 40, 60, 80, 100};
const float fuel_map_rpm[FUEL_MAP_N_RPM] = {500, 800, 1500, 2500, 3500, 4500, 6000, 8000};

/*
 * t_base = m_ar(TPS, RPM) / (AFR_st · Q_inj), em µs, com o modelo nominal de
 * docs/equacoes.md (bico novo, sem vazamento). Numa ECU real esta tabela vem
 * da calibração em dinamômetro. test_fuel_map.c confere os pontos.
 */
const float fuel_map_us[FUEL_MAP_N_TPS][FUEL_MAP_N_RPM] = {
    {  777,   801,   850,   898,   922,   922,   877,   732}, /* TPS   0 % */
    { 1476,  1523,  1614,  1706,  1752,  1752,  1666,  1390}, /* TPS  10 % */
    { 2176,  2244,  2379,  2514,  2582,  2582,  2455,  2049}, /* TPS  20 % */
    { 3574,  3686,  3908,  4131,  4242,  4242,  4033,  3366}, /* TPS  40 % */
    { 4973,  5129,  5437,  5747,  5902,  5902,  5611,  4683}, /* TPS  60 % */
    { 6372,  6571,  6967,  7363,  7562,  7562,  7190,  6000}, /* TPS  80 % */
    { 7770,  8013,  8496,  8980,  9221,  9221,  8768,  7317}, /* TPS 100 % */
};

/* Índice do intervalo [i, i+1] que contém x e a fração dentro dele. */
static int locate(const float *axis, int n, float x, float *frac)
{
    if(x <= axis[0]) {
        *frac = 0.0f;
        return 0;
    }
    if(x >= axis[n - 1]) {
        *frac = 1.0f;
        return n - 2;
    }
    int i = 0;
    while(x > axis[i + 1]) {
        i++;
    }
    *frac = (x - axis[i]) / (axis[i + 1] - axis[i]);
    return i;
}

float fuel_map_base_us(float tps, float rpm)
{
    float ft, fr;
    int i = locate(fuel_map_tps, FUEL_MAP_N_TPS, tps, &ft);
    int j = locate(fuel_map_rpm, FUEL_MAP_N_RPM, rpm, &fr);
    float a = fuel_map_us[i][j] + (fuel_map_us[i][j + 1] - fuel_map_us[i][j]) * fr;
    float b = fuel_map_us[i + 1][j] + (fuel_map_us[i + 1][j + 1] - fuel_map_us[i + 1][j]) * fr;
    return a + (b - a) * ft;
}
