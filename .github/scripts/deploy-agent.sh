#!/bin/bash


set -euo pipefail

eval `ssh-agent -s`

echo "${AGENT_SSH_KEY}" > key
ls -al
ssh-add - <<< "${AGENT_SSH_KEY}"
