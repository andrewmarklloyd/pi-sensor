#!/bin/bash


set -euo pipefail

# eval `ssh-agent -s`

echo "${AGENT_SSH_KEY}" > ~/.ssh/id
chmod 600 ~/.ssh/id
# ssh-add ~/.ssh/id
ssh -i ~/.ssh/id pi@${AGENT_HOST} exit
echo $?
