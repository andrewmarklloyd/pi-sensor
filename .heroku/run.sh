#!/bin/bash



curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.37.2/install.sh | bash
cd consumer/frontend
nvm use
npm run build
