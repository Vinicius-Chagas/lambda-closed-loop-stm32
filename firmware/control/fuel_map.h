/*
 * Tabela de combustível base (feedforward): tempo de injeção efetivo, sem o
 * tempo morto, que dá λ = 1 com o motor nominal. Interpolação bilinear.
 */
#ifndef FUEL_MAP_H
#define FUEL_MAP_H

#define FUEL_MAP_N_TPS 7
#define FUEL_MAP_N_RPM 8

extern const float fuel_map_tps[FUEL_MAP_N_TPS];
extern const float fuel_map_rpm[FUEL_MAP_N_RPM];
extern const float fuel_map_us[FUEL_MAP_N_TPS][FUEL_MAP_N_RPM];

/* t_base em µs; fora da tabela usa a borda (sem extrapolar). */
float fuel_map_base_us(float tps, float rpm);

#endif /* FUEL_MAP_H */
