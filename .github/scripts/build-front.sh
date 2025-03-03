#!/bin/bash

set -euo pipefail

cd frontend
echo "PUBLIC_REACT_APP_VERSION=${REACT_APP_VERSION}" > .env

newname="prod.service-worker.$(uuidgen | cut -c25-36).js"
sed -i "s/service-worker.js/${newname}/g" public/index.html
cp public/service-worker.js public/${newname}

npm install --include=optional
npm run build
