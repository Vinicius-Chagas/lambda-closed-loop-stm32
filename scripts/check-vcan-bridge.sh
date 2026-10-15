#!/usr/bin/env bash
# Verifica a ponte SocketCAN: sobe a ECU no Renode ligada ao vcan0 e espera
# frames de estado (0x100) com o candump. Requer vcan0 (scripts/setup-vcan.sh),
# can-utils e o firmware compilado.
set -euo pipefail
cd "$(dirname "$0")/.."

IMAGE=${RENODE_IMAGE:-antmicro/renode@sha256:b30df3a8ec4eb6c793cca6ed62733c3619363de52ffd9f52de98a11f3500471b}
NAME=ecu-vcan-check
IF=${1:-vcan0}

docker rm -f "$NAME" >/dev/null 2>&1 || true
trap 'docker logs "$NAME" > build/vcan-renode.log 2>&1 || true; docker rm -f "$NAME" >/dev/null 2>&1 || true' EXIT
mkdir -p build

docker run -d --name "$NAME" --network host -v "$PWD":/work -w /work "$IMAGE" \
    renode --disable-xwt --console --plain \
    -e "\$can_if=\"$IF\"; include @renode/lambda-vcan.resc; sysbus.adc.lambdaSensor Volts 1.65; start" >/dev/null

# 5 frames 0x100 em até 60 s (inclui a partida do Renode).
if ! timeout 60 candump -n 5 -L "$IF,100:7FF" | tee build/vcan-frames.log; then
    echo "FALHA: nenhum frame 0x100 em $IF" >&2
    exit 1
fi
echo "OK: frames da ECU chegaram em $IF"
