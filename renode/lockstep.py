# -*- coding: utf-8 -*-
# Comando de monitor usado pela planta (go/internal/renode) para avançar a
# emulação em um único comando por passo, em vez de seis:
#   lockstep <V lambda> <V tps> <V rpm> <dt em s>
# Ajusta as tensões dos canais do ADC, roda dt de tempo virtual e imprime
# "CCR1 ARR" do TIM2 (largura do pulso e período do injetor, em µs).
from System import Decimal

_SENSORES = ("sysbus.adc.lambdaSensor", "sysbus.adc.tpsSensor", "sysbus.adc.rpmSensor")
_TIM2_CCR1 = 0x40000034
_TIM2_ARR = 0x4000002C


def mc_lockstep(v_lambda, v_tps, v_rpm, dt):
    machine = monitor.Machine
    for nome, v in zip(_SENSORES, (v_lambda, v_tps, v_rpm)):
        machine[nome].Volts = Decimal(float(v))
    monitor.Parse('emulation RunFor "%s"' % dt)
    bus = machine.SystemBus
    print("0x%08X 0x%08X" % (bus.ReadDoubleWord(_TIM2_CCR1), bus.ReadDoubleWord(_TIM2_ARR)))
