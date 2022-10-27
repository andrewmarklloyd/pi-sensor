#!/bin/bash

set -euo pipefail

eval `ssh-agent -s`
ssh-add - <<< "${AGENT_SSH_KEY}"
ssh -o StrictHostKeyChecking=no pi@${AGENT_HOST} uptime
