#!/bin/bash


set -euo pipefail

ssh-add - <<< "${AGENT_SSH_KEY}"
