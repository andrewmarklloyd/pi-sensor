#!/bin/bash


set -euo pipefail

eval `ssh-agent -s`

echo "$SSH_PRIVATE_KEY" | ssh-add -
# echo "${AGENT_SSH_KEY}" > key
# ssh-add key
