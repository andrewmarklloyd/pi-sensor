#!/bin/bash

set -euo pipefail

ssh-add - <<< "${AGENT_SSH_KEY}"
ssh -o StrictHostKeyChecking=no pi@${AGENT_HOST} uptime
