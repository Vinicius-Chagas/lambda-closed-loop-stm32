*** Settings ***
Documentation     Integração do firmware da ECU rodando no Renode (STM32F072).
...               Rodar com: make test-renode
Library           ${CURDIR}/can_frames.py
Test Setup        Iniciar ECU
Test Teardown     Test Teardown

*** Variables ***
${ROOT}           ${CURDIR}/../..
${ELF}            ${ROOT}/firmware/build/ecu.elf
${REPL}           ${ROOT}/renode/stm32f072_lambda.repl
${UART}           sysbus.usart1
${TIM2_ARR}       0x4000002C
${TIM2_CCR1}      0x40000034
${ID_ESTADO}      0x100
${ID_SENSORES}    0x101
${ID_TEMPOS}      0x110
${ID_PAINEL}      0x200

*** Keywords ***
Iniciar ECU
    Execute Command           $bin=@${ELF}
    Execute Command           $repl=@${REPL}
    Execute Command           include @${ROOT}/renode/lambda.resc
    Create Terminal Tester    ${UART}    defaultPauseEmulation=true

Ajustar Sensores
    [Arguments]    ${lambda_v}    ${tps_v}    ${rpm_v}
    Execute Command    sysbus.adc.lambdaSensor Volts ${lambda_v}
    Execute Command    sysbus.adc.tpsSensor Volts ${tps_v}
    Execute Command    sysbus.adc.rpmSensor Volts ${rpm_v}

Ler Registrador
    [Arguments]    ${endereco}
    ${valor}=    Execute Command    sysbus ReadDoubleWord ${endereco}
    ${valor}=    Convert To Integer    ${valor.strip()}
    RETURN    ${valor}

Contagem Do ADC
    [Arguments]    ${volts}
    ${contagem}=    Evaluate    round(${volts} / 3.3 * 4095)
    RETURN    ${contagem}

Esperar ECU Com Sensores Estaveis
    [Documentation]    Sobe a ECU com tensões fixas e espera o 1º log com as médias certas.
    Ajustar Sensores    1.65    0.5    3.0
    Wait For Line On Uart    ECU pronta
    Wait For Line On Uart    ADC lambda=2048 tps=620 rpm=3723    timeout=1

Ir Para 2500 rpm E 20 Por Cento
    [Documentation]    Ponto da tabela de combustível: 0,84 V → 20 % TPS; 1,03 V → 2500 rpm.
    [Arguments]    ${lambda_v}=1.65
    Ajustar Sensores    ${lambda_v}    0.84    1.03125
    # Regex com "." no lugar de "=": o servidor do Renode lê "x=" como argumento nomeado.
    Wait For Line On Uart    tps.1042 rpm.1279    timeout=1    treatAsRegex=true

Enviar Modo
    [Documentation]    0 = malha aberta, 1 = malha fechada (λ alvo mantido).
    [Arguments]    ${modo}
    Execute Command    can_send ${ID_PAINEL} "00000${modo}00"
    Wait For Line On Uart    CMD alvo=0 modo=${modo} flags=0    timeout=0.1

*** Test Cases ***
Deve Inicializar O Firmware
    Wait For Line On Uart    ECU lambda - boot
    Wait For Line On Uart    ECU pronta

ADC Deve Ler As Tensoes Injetadas Via DMA
    Ajustar Sensores    1.65    0.5    3.0
    ${l}=    Contagem Do ADC    1.65
    ${t}=    Contagem Do ADC    0.5
    ${r}=    Contagem Do ADC    3.0
    Wait For Line On Uart    ADC lambda=${l} tps=${t} rpm=${r}    timeout=1

    Ajustar Sensores    0.6    2.0    1.2
    ${l}=    Contagem Do ADC    0.6
    ${t}=    Contagem Do ADC    2.0
    ${r}=    Contagem Do ADC    1.2
    Wait For Line On Uart    ADC lambda=${l} tps=${t} rpm=${r}    timeout=1

Injetor Deve Ter Um Pulso Por Ciclo Do Motor
    Esperar ECU Com Sensores Estaveis
    # 3,0 V no conversor F/V → 3723 contagens → 7273 rpm → ciclo de 120/rpm s.
    ${ciclo}=    Evaluate    round(120e6 / (3723 / 4095 * 8000))
    ${arr}=      Ler Registrador    ${TIM2_ARR}
    Should Be Equal As Integers    ${arr}    ${ciclo - 1}

    # Malha aberta em λ = 1: pulso = tempo morto + tabela (±0,5 % do modelo).
    Enviar Modo    0
    Ir Para 2500 rpm E 20 Por Cento
    ${ccr}=      Ler Registrador    ${TIM2_CCR1}
    ${ti}=       Tempo De Injecao Nominal    20    2500
    Should Be True    abs(${ccr} - ${ti}) <= 0.005 * ${ti}    CCR1=${ccr} esperado=${ti}

    # Marcha lenta: 800 rpm → ciclo de 150 ms (não cabe em 16 bits: TIM2 de 32 bits).
    Ajustar Sensores    1.65    0.3    0.33
    Wait For Line On Uart    ADC lambda=2048 tps=372 rpm=410    timeout=1
    ${ciclo}=    Evaluate    round(120e6 / (410 / 4095 * 8000))
    ${arr}=      Ler Registrador    ${TIM2_ARR}
    Should Be Equal As Integers    ${arr}    ${ciclo - 1}

Malha Fechada Deve Corrigir A Mistura Ate O Limite
    Esperar ECU Com Sensores Estaveis
    Enviar Modo    0
    Ir Para 2500 rpm E 20 Por Cento
    ${ti0}=      Ler Registrador    ${TIM2_CCR1}

    # Sonda em λ = 1,10 (2,2 V): mistura pobre → correção sobe até +30 %.
    Enviar Modo    1
    Ir Para 2500 rpm E 20 Por Cento    2.2
    Execute Command    emulation RunFor "2"
    ${ti}=       Ler Registrador    ${TIM2_CCR1}
    Should Be True    abs(${ti} - (800 + (${ti0} - 800) * 1.30)) <= 2    ti=${ti} ti0=${ti0}

    # Sonda em λ = 0,90 (1,1 V): mistura rica → correção desce até −30 %.
    Ir Para 2500 rpm E 20 Por Cento    1.1
    Execute Command    emulation RunFor "4"
    ${ti}=       Ler Registrador    ${TIM2_CCR1}
    Should Be True    abs(${ti} - (800 + (${ti0} - 800) * 0.70)) <= 2    ti=${ti} ti0=${ti0}

    # De volta à malha aberta: só a tabela.
    Enviar Modo    0
    Execute Command    emulation RunFor "0.05"
    ${ti}=       Ler Registrador    ${TIM2_CCR1}
    Should Be Equal As Integers    ${ti}    ${ti0}

ECU Deve Publicar Estado E Sensores No CAN
    Esperar ECU Com Sensores Estaveis
    Create CAN Tester    canHub    defaultTimeout=1
    ${dados}=    Wait For Frame With Id    ${ID_ESTADO}
    ${estado}=   Decodificar Estado    ${dados}
    Should Be Equal As Integers    ${estado}[lambda_med]     1000
    Should Be Equal As Integers    ${estado}[lambda_alvo]    1000
    Should Be Equal As Integers    ${estado}[modo]           1
    ${ccr}=      Ler Registrador    ${TIM2_CCR1}
    Should Be Equal As Integers    ${estado}[ti_us]          ${ccr}

    ${dados}=    Wait For Frame With Id    ${ID_SENSORES}
    ${sens}=     Decodificar Sensores    ${dados}
    Should Be Equal As Integers    ${sens}[rpm]       7273
    Should Be Equal As Integers    ${sens}[tps_x10]   74

ECU Deve Aceitar Comando Do Painel
    Esperar ECU Com Sensores Estaveis
    # λ alvo 1,200 (0x04B0), malha aberta, zera estatísticas de tempo.
    Execute Command    can_send ${ID_PAINEL} "B0040001"
    Wait For Line On Uart    CMD alvo=1200 modo=0 flags=1    timeout=0.1
    Create CAN Tester    canHub    defaultTimeout=1
    ${dados}=    Wait For Frame With Id    ${ID_ESTADO}
    ${estado}=   Decodificar Estado    ${dados}
    Should Be Equal As Integers    ${estado}[lambda_alvo]    1200
    Should Be Equal As Integers    ${estado}[modo]           0

    # Valores fora da faixa são ignorados; 0xFF mantém o modo.
    Execute Command    can_send ${ID_PAINEL} "D007FF00"
    Wait For Line On Uart    CMD alvo=2000 modo=255 flags=0    timeout=0.1
    Create CAN Tester    canHub    defaultTimeout=1
    ${dados}=    Wait For Frame With Id    ${ID_ESTADO}
    ${estado}=   Decodificar Estado    ${dados}
    Should Be Equal As Integers    ${estado}[lambda_alvo]    1200
    Should Be Equal As Integers    ${estado}[modo]           0

ControlTask Deve Cumprir Os Requisitos De Tempo Real
    Esperar ECU Com Sensores Estaveis
    # Zera as estatísticas depois da partida e mede por 2 s de tempo simulado.
    Execute Command    can_send ${ID_PAINEL} "0000FF01"
    Wait For Line On Uart    CMD alvo=0 modo=255 flags=1    timeout=0.1
    Execute Command    emulation RunFor "2"
    Create CAN Tester    canHub    defaultTimeout=1
    ${dados}=    Wait For Frame With Id    ${ID_TEMPOS}
    ${t}=        Decodificar Tempos    ${dados}
    Log    ${t}    console=true
    Should Be True    abs(${t}[periodo_us] - 10000) <= 100
    Should Be True    ${t}[jitter_max_us] <= 100
    Should Be True    0 < ${t}[exec_max_us] <= 1000
    Should Be Equal As Integers    ${t}[deadlines_perdidos]    0
