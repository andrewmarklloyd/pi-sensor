#!/bin/bash


set -euo pipefail

if ! command -v yq &> /dev/null; then
  wget https://github.com/mikefarah/yq/releases/download/v4.27.5/yq_linux_amd64.tar.gz -O - |\
  tar xz && mv yq_linux_amd64 /usr/bin/yq
fi

if ! command -v doctl &> /dev/null; then
  doctlVersion="1.119.1"
  wget -q https://github.com/digitalocean/doctl/releases/download/v${doctlVersion}/doctl-${doctlVersion}-linux-amd64.tar.gz -P /tmp
  tar xf /tmp/doctl-${doctlVersion}-linux-amd64.tar.gz -C /tmp
  mv /tmp/doctl /usr/local/bin
fi

deploy() {
  echo "Deploying version ${SHORT_SHA}"

  doctl --access-token ${DO_ACCESS_TOKEN} auth init || (echo "doctl not authenticated" && exit 1)
  DO_APP_ID=$(doctl --access-token ${DO_ACCESS_TOKEN} apps list -o json | yq -r '.[] | select(.spec.name == "pi-sensor").id')

  TFILE=$(mktemp --suffix .yaml)
  doctl --access-token ${DO_ACCESS_TOKEN} apps spec get ${DO_APP_ID} > ${TFILE}
  yq -i ".services[0].image.tag = \"${SHORT_SHA}\"" ${TFILE}
  doctl --access-token ${DO_ACCESS_TOKEN} apps update ${DO_APP_ID} --wait --spec "${TFILE}"
  rm -f ${TFILE}
}

git diff ':!frontend/package-lock.json' ':!frontend/public/index.html' ':!frontend/public/*service-worker*'
SHORT_SHA=$(echo ${GITHUB_SHA} | cut -c1-7)
deploy
