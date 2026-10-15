# Equações e parâmetros do modelo

Este documento define a física simulada pela planta (Go) e as contas feitas pela ECU (firmware C).
Os valores numéricos aqui são os mesmos usados no código; se mudar um, mude o outro
(`go/internal/engine/params.go` e `firmware/control/params.h`). O teste
`go/internal/sil` compara as duas implementações.

## 1. Fator lambda

A relação ar/combustível estequiométrica da gasolina pura é **AFR_st = 14,7** (kg de ar por kg de
combustível). O fator lambda normaliza a mistura real por esse valor:

```
λ = (m_ar / m_comb) / AFR_st
```

| λ | Mistura | Efeito |
|---|---|---|
| < 1 | rica (sobra combustível) | mais potência, mais CO e HC, mais consumo |
| = 1 | estequiométrica | ponto de máxima eficiência do catalisador |
| > 1 | pobre (sobra ar) | mais NOx, risco de falha de combustão |

> A gasolina vendida no Brasil (tipo C, com ~27 % de etanol) tem AFR_st ≈ 13,2. Como o controle
> trabalha em λ, trocar o combustível só muda a constante `AFR_ST`; o resto do modelo vale igual.

## 2. Motor (planta)

Motor de 4 cilindros, 4 tempos, 1,6 L.

| Símbolo | Valor | Descrição |
|---|---|---|
| `V_cil` | 0,4 L | volume de um cilindro |
| `ρ_ar` | 1,2 kg/m³ | densidade do ar na admissão |
| `m_ar_max` | ρ_ar · V_cil = 480 mg | massa de ar que cabe no cilindro |
| `RPM_lenta` | 800 rpm | marcha lenta |
| `RPM_max` | 6000 rpm | rotação com borboleta toda aberta |
| `τ_rpm` | 0,5 s | constante de tempo da rotação |

**Rotação.** A bancada não simula carga; a rotação segue a borboleta com uma dinâmica de 1ª ordem:

```
RPM_alvo = RPM_lenta + (RPM_max − RPM_lenta) · TPS/100
dRPM/dt = (RPM_alvo − RPM) / τ_rpm
```

**Ar admitido por cilindro, por ciclo** (modelo alfa-N: borboleta + rotação):

```
f(TPS)   = 0,10 + 0,90 · TPS/100                 (0,10 = ar que passa com a borboleta fechada)
η(RPM)   = 0,85 − 0,10 · ((RPM − 4000) / 3000)²  (eficiência volumétrica)
m_ar     = m_ar_max · f(TPS) · η(RPM) + m_vazamento
```

`m_vazamento` é a perturbação "vazamento de ar": ar que entra depois do ponto de medição e
que a ECU não enxerga.

**Combustível injetado por cilindro, por ciclo:**

| Símbolo | Valor | Descrição |
|---|---|---|
| `Q_inj` | 3,0 mg/ms | vazão do bico (≈ 245 cm³/min de gasolina) |
| `t_morto` | 0,8 ms | tempo de abertura do bico, sem vazão |

```
m_comb = k_inj · Q_inj · max(0, t_i − t_morto)
```

`k_inj` é a perturbação "bico sujo/desgastado": 1,0 no bico novo, 0,8 = entrega 20 % a menos.

**Lambda no cilindro:**

```
λ_cil = m_ar / (m_comb · AFR_st)
```

## 3. Atraso de transporte (escape)

O gás leva um tempo para sair do cilindro e chegar à sonda. Parte do atraso é proporcional ao
ciclo do motor (escape e mistura no coletor), parte é fixa (percurso no tubo):

```
T_d = 180 / RPM + 0,020   [s]
```

| Rotação | T_d |
|---|---|
| 800 rpm (lenta) | 245 ms |
| 2000 rpm | 110 ms |
| 6000 rpm | 50 ms |

Na planta, `λ_cil` é gravado num buffer circular a cada passo da simulação; a sonda lê o valor
de `T_d` segundos atrás (`λ_escape(t) = λ_cil(t − T_d)`).

## 4. Sonda lambda (banda larga)

Resposta de 1ª ordem com **τ_sonda = 100 ms** (sondas banda larga tipo LSU respondem em menos de
100 ms; sondas banda estreita são mais lentas):

```
dλ_sonda/dt = (λ_escape − λ_sonda) / τ_sonda
```

Saída elétrica linear, já condicionada para a faixa do ADC:

```
V_sonda = 3,3 · (λ_sonda − 0,7) / 0,6      saturada em [0; 3,3] V
```

| λ | V_sonda |
|---|---|
| 0,70 | 0,00 V |
| 1,00 | 1,65 V |
| 1,30 | 3,30 V |

## 5. Demais sensores

| Sensor | Equação | Faixa |
|---|---|---|
| TPS | `V = 0,3 + 2,7 · TPS/100` | 0,3 – 3,0 V |
| Rotação (conversor F/V) | `V = 3,3 · RPM / 8000` | 0 – 3,3 V |

O ADC é de 12 bits com referência de 3,3 V: `contagem = round(V / 3,3 · 4095)`.

## 6. ECU: cálculo do tempo de injeção

A cada 10 ms a ECU:

1. converte as médias do ADC em `λ_med`, `TPS` e `RPM` (inverso das equações acima);
2. busca o tempo de combustível base numa tabela 2D `t_base(TPS, RPM)` com interpolação bilinear
   (feedforward);
3. aplica a correção de malha fechada `c`:

```
t_i = t_morto + t_base · (1 + c)
```

A tabela foi calculada com o modelo nominal (bico novo, sem vazamento) para dar λ = 1:

```
t_base = m_ar(TPS, RPM) / (AFR_st · Q_inj)
```

### Controlador PI

```
e = λ_med − λ_alvo                (e > 0: mistura pobre, precisa de mais combustível)
c = Kp · e + I
I ← I + Ki · e · Δt               (só integra se c não estiver saturado no mesmo sentido do erro)
c saturado em ±0,25
```

| Parâmetro | Valor |
|---|---|
| `Kp` | 0,15 |
| `Ki` | 2,0 1/s |
| `Δt` | 10 ms |
| limite de `c` | ±25 % |

Ajuste: perto de λ = 1 a planta se comporta como `λ ≈ 1/(1+c)` (ganho ≈ −1), com atraso
`T_d` + `τ_sonda`. Os ganhos seguem a regra SIMC para o pior caso (marcha lenta, atraso de 245 ms):
`Kc = τ / (K·(τc + θ))` com `τc = θ`, ligeiramente reduzido para manter folga em todas as rotações.
O valor final é confirmado pelo teste SIL (`go/internal/sil`).

**Malha aberta:** `c = 0` e o integrador é zerado; o tempo de injeção vem só da tabela.

## 7. Tempo do ciclo e pulso do injetor

Um pulso por ciclo do motor (2 voltas):

```
T_ciclo = 120 / RPM   [s]   → 150 ms em lenta, 20 ms a 6000 rpm
```

O TIM2 da ECU gera PWM com período `T_ciclo` e largura `t_i`, ambos em µs.
