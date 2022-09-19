#!/bin/bash


set -euo pipefail

if ! command -v yq &> /dev/null; then
  wget https://github.com/mikefarah/yq/releases/download/v4.27.5/yq_linux_amd64.tar.gz -O - |\
  tar xz && mv yq_linux_amd64 /usr/bin/yq
fi

if ! command -v doctl &> /dev/null; then
  cd ~/
  wget https://github.com/digitalocean/doctl/releases/download/v1.79.0/doctl-1.79.0-linux-amd64.tar.gz
  tar xf ~/doctl-1.79.0-linux-amd64.tar.gz
  mv ~/doctl /usr/local/bin
  doctl auth init --access-token ${DO_ACCESS_TOKEN}
fi

deploy() {
  echo "Deploying version ${SHORT_SHA}"
  doctl apps spec get ${APP_ID} | yq ".services[0].image.tag = \"${SHORT_SHA}\"" - #| doctl apps update ${APP_ID} --spec -
}

git diff
SHORT_SHA=$(echo ${GITHUB_SHA} | cut -c1-7)
app=${1}
deploy
