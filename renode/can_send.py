# Comando de monitor para injetar um frame CAN no controlador da ECU, sem
# precisar de vcan0 (usado nos testes Robot):
#   can_send 0x200 "E8030100"
from Antmicro.Renode.Core.CAN import CANMessageFrame
from System import Array, Byte


def mc_can_send(can_id, hex_data, peripheral="sysbus.can"):
    data = bytearray.fromhex(str(hex_data))
    frame = CANMessageFrame(int(can_id), Array[Byte](list(data)))
    monitor.Machine[peripheral].OnFrameReceived(frame)
