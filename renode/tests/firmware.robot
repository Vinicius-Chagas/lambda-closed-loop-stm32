*** Settings ***
Documentation     Integração do firmware da ECU rodando no Renode (STM32F072).
...               Rodar com: renode-test renode/tests/firmware.robot
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

Injetor Deve Refletir O Pulso Calculado
    # Prova de conceito: ciclo fixo de 60 ms e pulso = tensão da sonda em mV.
    Ajustar Sensores    1.65    0.5    3.0
    # O servidor do Renode lê "nome=valor" como argumento nomeado; casamos só o número.
    Wait For Line On Uart    1650 us    timeout=1
    ${arr}=     Ler Registrador    ${TIM2_ARR}
    ${ccr}=     Ler Registrador    ${TIM2_CCR1}
    Should Be Equal As Integers    ${arr}    59999
    Should Be Equal As Integers    ${ccr}    1650

ECU Deve Publicar Estado No CAN
    Ajustar Sensores    1.65    0.5    3.0
    Create CAN Tester    canHub    defaultTimeout=1
    Wait For Line On Uart    ECU pronta
    # Descarta frames anteriores ao ajuste dos sensores.
    Wait For Line On Uart    ADC lambda=2048    timeout=1
    ${dados}=    Wait For Frame With Id    ${ID_ESTADO}    timeout=0.1
    Length Should Be    ${dados}    6
    ${lambda}=    Evaluate    int.from_bytes(bytes($dados)[0:2], 'little')
    ${tps}=       Evaluate    int.from_bytes(bytes($dados)[2:4], 'little')
    ${rpm}=       Evaluate    int.from_bytes(bytes($dados)[4:6], 'little')
    Should Be Equal As Integers    ${lambda}    2048
    Should Be Equal As Integers    ${tps}       620
    Should Be Equal As Integers    ${rpm}       3723
