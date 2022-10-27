#!/bin/bash


set -euo pipefail

curl -sSfo op.zip https://cache.agilebits.com/dist/1P/op2/pkg/v2.7.1/op_linux_amd64_v2.7.1.zip
unzip -od /usr/local/bin/ op.zip
rm op.zip

/usr/local/bin/op document get pi-sensor-agent-ssh-key-id_ed25519 > id_ed25519
chmod 600 id_ed25519
ssh -o StrictHostKeyChecking=no -i id_ed25519 pi@${AGENT_HOST} uptime
rm id_ed25519
