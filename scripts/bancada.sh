#!/usr/bin/env bash
# Sobe a bancada: ECU (firmware no Renode, em Docker) + planta (Go, nativa).
# Argumentos extras vão para a planta, ex.:
#   scripts/bancada.sh -csv build/corrida.csv
#   scripts/bancada.sh -can ""            # sem vcan0 (só a planta e a ECU)
# Ctrl+C encerra as duas. Requer o firmware compilado (make firmware) e, com
# CAN, o vcan0 (scripts/setup-vcan.sh).
set -euo pipefail
cd "$(dirname "$0")/.."

IMAGE=${RENODE_IMAGE:-antmicro/renode@sha256:b30df3a8ec4eb6c793cca6ed62733c3619363de52ffd9f52de98a11f3500471b}
NAME=${BANCADA_NOME:-lambda-ecu}
PORT=${RENODE_PORT:-1234}
LOG=${BANCADA_LOG:-build/bancada-renode.log}

[[ -f firmware/build/ecu.elf ]] || { echo "falta firmware/build/ecu.elf: rode make firmware (ou make firmware-docker)" >&2; exit 1; }

# Interface CAN que a planta vai usar (padrão da planta: vcan0).
can=vcan0
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
    case ${args[i]} in
        -can | --can) can=${args[i + 1]-} ;;
        -can=* | --can=*) can=${args[i]#*=} ;;
    esac
done
if [[ -n "$can" ]] && ! ip link show "$can" >/dev/null 2>&1; then
    echo "interface $can não existe: rode ./scripts/setup-vcan.sh (refazer após reiniciar)" >&2
    exit 1
fi
mkdir -p build "$(dirname "$LOG")"
(cd go && go build -o ../build/planta ./cmd/planta)

planta=""
parar() {
    trap - EXIT INT TERM
    [[ -n "$planta" ]] && kill "$planta" 2>/dev/null && wait "$planta" 2>/dev/null || true
    docker logs "$NAME" > "$LOG" 2>&1 || true
    docker rm -f "$NAME" >/dev/null 2>&1 || true
}
trap parar EXIT INT TERM

docker rm -f "$NAME" >/dev/null 2>&1 || true
# --network host: o Renode enxerga o vcan0 do host. -i: sem stdin o Renode encerra.
docker run -d -i --name "$NAME" --network host -v "$PWD":/work -w /work "$IMAGE" \
    renode --disable-xwt --plain --port "$PORT" >/dev/null

# A planta carrega o script da ECU pelo monitor e conduz a simulação.
./build/planta -renode "localhost:$PORT" "$@" &
planta=$!
wait "$planta"
