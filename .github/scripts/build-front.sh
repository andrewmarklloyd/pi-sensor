#!/bin/bash

set -euo pipefail

cd frontend

newname="prod.service-worker.$(uuidgen | cut -c25-36).js"
sed -i "s/service-worker.js/${newname}/g" public/index.html
cp public/service-worker.js public/${newname}

npm install
npm run build
