#!/usr/bin/env bash
# Cria a interface CAN virtual vcan0 (precisa de root).
# Arch, Fedora, Debian/Ubuntu: o módulo vcan vem no kernel padrão; no Ubuntu
# de nuvem pode ser preciso instalar linux-modules-extra-$(uname -r).
set -euo pipefail
IF=${1:-vcan0}

if ! ip link show "$IF" >/dev/null 2>&1; then
    sudo modprobe vcan
    sudo ip link add dev "$IF" type vcan
fi
sudo ip link set up "$IF"
ip -details link show "$IF"
