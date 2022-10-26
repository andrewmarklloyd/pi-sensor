#!/bin/bash


set -euo pipefail

curl -sSfo op.zip https://cache.agilebits.com/dist/1P/op2/pkg/v2.7.1/op_linux_amd64_v2.7.1.zip
unzip -od /usr/local/bin/ op.zip
rm op.zip

mkdir -p ~/.ssh/
/usr/local/bin/op read op://github-ci/pi-sensor-agent-ssh-key/private\ key > ~/.ssh/id
chmod 600 ~/.ssh/id
ssh -o StrictHostKeyChecking=no -i ~/.ssh/id pi@${AGENT_HOST} uptime
echo $?
