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

# -i mantém o stdin aberto: sem ele o console do Renode recebe EOF e encerra.
docker run -d -i --name "$NAME" --network host -v "$PWD":/work -w /work "$IMAGE" \
    renode --disable-xwt --console --plain \
    -e "\$can_if=\"$IF\"; include @renode/lambda-vcan.resc; sysbus.adc.lambdaSensor Volts 1.65; start" >/dev/null

# 5 frames 0x100 em até 60 s (inclui a partida do Renode).
# Mostra também os demais IDs (útil para diagnóstico).
timeout 30 candump -L "$IF" > build/vcan-todos.log 2>&1 &
if ! timeout 60 candump -n 20 -L "$IF,100:7FF" | tee build/vcan-frames.log; then
    echo "FALHA: menos de 20 frames 0x100 em $IF" >&2
    echo "--- log do Renode ---" >&2
    docker logs "$NAME" 2>&1 | tail -60 >&2
    echo "--- todos os frames em $IF ---" >&2
    head -40 build/vcan-todos.log >&2 || true
    exit 1
fi
echo "OK: frames da ECU chegaram em $IF"
