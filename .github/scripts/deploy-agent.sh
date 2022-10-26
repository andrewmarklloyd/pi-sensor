#!/bin/bash


set -euo pipefail

eval `ssh-agent -s`

echo "$AGENT_SSH_KEY" | ssh-add -
# echo "${AGENT_SSH_KEY}" > key
# ssh-add key
