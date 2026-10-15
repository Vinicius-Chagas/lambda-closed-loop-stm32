#!/usr/bin/env bash
# Teste ponta a ponta: sobe a bancada no vcan0 e roda go/e2e.
# Requer vcan0 (scripts/setup-vcan.sh), Docker, Go e o firmware compilado.
# Resultados em build/e2e: CSV da corrida, log da planta e do Renode.
set -euo pipefail
cd "$(dirname "$0")/.."
OUT=build/e2e
mkdir -p "$OUT"

BANCADA_NOME=lambda-e2e BANCADA_LOG=$OUT/renode.log \
    scripts/bancada.sh -can vcan0 -csv "$OUT/corrida.csv" > "$OUT/planta.log" 2>&1 &
bancada=$!
trap 'kill $bancada 2>/dev/null; wait $bancada 2>/dev/null || true' EXIT

status=0
(cd go && go test -tags e2e -count=1 -v -timeout 15m ./e2e -args -can vcan0) || status=$?
if [[ $status -ne 0 ]]; then
    echo "--- planta (últimas linhas) ---" >&2
    tail -40 "$OUT/planta.log" >&2 || true
fi
exit $status
