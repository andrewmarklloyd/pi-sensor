#!/bin/bash


set -euo pipefail

if ! command -v yq &> /dev/null; then
  wget https://github.com/mikefarah/yq/releases/download/v4.27.5/yq_linux_amd64.tar.gz -O - |\
  tar xz && mv yq_linux_amd64 /usr/bin/yq
fi

if ! command -v doctl &> /dev/null; then
  wget -q https://github.com/digitalocean/doctl/releases/download/v1.84.1/doctl-1.84.1-linux-amd64.tar.gz -P /tmp
  tar xf /tmp/doctl-1.84.1-linux-amd64.tar.gz -C /tmp
  mv /tmp/doctl /usr/local/bin
fi

deploy() {
  echo "Deploying version ${SHORT_SHA}"
  doctl --access-token ${DO_ACCESS_TOKEN} registry login --expiry-seconds 300
  image="registry.digitalocean.com/pi-sensor/pi-sensor:${SHORT_SHA}"
  echo "${OPCONNECT_CERT}" > opconnect.crt
  docker build -f ./docker/do.Dockerfile -t ${image} .
  docker push ${image}
  doctl --access-token ${DO_ACCESS_TOKEN} apps spec get ${DO_APP_ID} | yq ".services[0].image.tag = \"${SHORT_SHA}\"" - | doctl --access-token ${DO_ACCESS_TOKEN} apps update ${DO_APP_ID} --wait --spec -
}

git diff
SHORT_SHA=$(echo ${GITHUB_SHA} | cut -c1-7)
deploy
