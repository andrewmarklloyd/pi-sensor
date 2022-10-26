#!/bin/bash


set -euo pipefail

eval `ssh-agent -s`

echo "${AGENT_SSH_KEY}" > key
tail -1 key
ssh-add key
