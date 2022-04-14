#!/bin/bash

set -euo pipefail

curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.37.2/install.sh | bash ;\
export NVM_DIR="${HOME}/.nvm"
source "${NVM_DIR}/nvm.sh"
cd server/frontend
nvm install
nvm use
npm install
npm run build
