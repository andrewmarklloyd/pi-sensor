#!/bin/bash


set -euo pipefail


privateKey=$(op read op://github-ci/pi-sensor-agent-ssh-key/private\ key)
ssh-add - <<< "${privateKey}"

