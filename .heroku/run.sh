#!/bin/bash



curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.37.2/install.sh | bash
. $HOME/.nvm/nvm.sh
cd consumer/frontend
nvm install
nvm use
npm install
npm run build


wget -c https://golang.org/dl/go1.15.2.linux-amd64.tar.gz
shasum -a 256 go1.15.2.linux-amd64.tar.gz
mkdir -p ./usr/local
tar -C ./usr/local -xvzf go1.15.2.linux-amd64.tar.gz


export GOPATH=$PWD/go
./usr/local/go/bin/go mod tidy
