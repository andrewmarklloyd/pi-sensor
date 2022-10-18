#!/bin/bash

set -euo pipefail

if ! command -v nvm &> /dev/null; then
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.37.2/install.sh | bash
    export NVM_DIR="${HOME}/.nvm"
    source "${NVM_DIR}/nvm.sh"
    cd frontend
    nvm install
    nvm use
fi
npm install
npm run build
