"""Decodificação dos frames CAN da ECU para os testes Robot (ver can/lambda.dbc)."""
import struct


def _bytes(dados):
    if hasattr(dados, "data"):  # xmlrpc.client.Binary
        dados = dados.data
    return bytes(bytearray(dados))


def decodificar_estado(dados):
    lambda_med, lambda_alvo, ti_us, correcao, flags = struct.unpack("<HHHbB", _bytes(dados)[:8])
    return {
        "lambda_med": lambda_med,
        "lambda_alvo": lambda_alvo,
        "ti_us": ti_us,
        "correcao_x400": correcao,
        "modo": flags & 0x01,
        "contador": flags >> 4,
    }


def decodificar_sensores(dados):
    rpm, tps_x10 = struct.unpack("<HH", _bytes(dados)[:4])
    return {"rpm": rpm, "tps_x10": tps_x10}


def decodificar_tempos(dados):
    periodo, jitter, exec_max, perdidos = struct.unpack("<HHHH", _bytes(dados)[:8])
    return {
        "periodo_us": periodo,
        "jitter_max_us": jitter,
        "exec_max_us": exec_max,
        "deadlines_perdidos": perdidos,
    }
