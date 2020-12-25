#!/bin/bash


curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.37.2/install.sh | bash
. $HOME/.nvm/nvm.sh
cd consumer/frontend
nvm install
nvm use
npm install
npm run build
