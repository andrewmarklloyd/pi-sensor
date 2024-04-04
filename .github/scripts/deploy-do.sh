#!/bin/bash


set -euo pipefail

if ! command -v yq &> /dev/null; then
  wget https://github.com/mikefarah/yq/releases/download/v4.27.5/yq_linux_amd64.tar.gz -O - |\
  tar xz && mv yq_linux_amd64 /usr/bin/yq
fi

if ! command -v doctl &> /dev/null; then
  doctlVersion="1.94.0"
  wget -q https://github.com/digitalocean/doctl/releases/download/v${doctlVersion}/doctl-${doctlVersion}-linux-amd64.tar.gz -P /tmp
  tar xf /tmp/doctl-${doctlVersion}-linux-amd64.tar.gz -C /tmp
  mv /tmp/doctl /usr/local/bin
fi

deploy() {
  echo "Deploying version ${SHORT_SHA}"
  doctl --access-token ${DO_ACCESS_TOKEN} registry login --expiry-seconds 300
  image="registry.digitalocean.com/pi-sensor/pi-sensor:${SHORT_SHA}"
  docker build -t ${image} .
  docker push ${image}
  doctl --access-token ${DO_ACCESS_TOKEN} apps spec get ${DO_APP_ID} | yq ".services[0].image.tag = \"${SHORT_SHA}\"" - | doctl --access-token ${DO_ACCESS_TOKEN} apps update ${DO_APP_ID} --wait --spec -
}

cleanup_tags() {
  maxTags=5
  tags=$(doctl --access-token ${DO_ACCESS_TOKEN} registry repo list-tags pi-sensor -o json)
  num=$(echo ${tags} | jq 'length')
  if [ "${num}" -gt "${maxTags}" ]; then
    diff=$((${num}-${maxTags}))
    echo "deleting oldest ${diff} tags from the container registry"
    toDelete=$(echo "${tags}" | jq -r '.[].tag' | tail -${diff})
    doctl --access-token ${DO_ACCESS_TOKEN} registry repo delete-tag pi-sensor --force ${toDelete}
  fi

  untaggedManifests=$(doctl --access-token ${DO_ACCESS_TOKEN} registry repo list-manifests pi-sensor -o json | jq -r '.[] | select(.tags | length==0) | .digest')
  if [[ ! -z ${untaggedManifests} ]]; then
    num=$(echo "${untaggedManifests}" | wc -l)
    echo "deleting ${num} untagged manifests from the container registry"
    doctl --access-token ${DO_ACCESS_TOKEN} registry repo delete-manifest pi-sensor ${untaggedManifests} --force
  fi
}

git diff ':!frontend/package-lock.json' ':!frontend/public/index.html' ':!frontend/public/*service-worker*'
SHORT_SHA=$(echo ${GITHUB_SHA} | cut -c1-7)
deploy
cleanup_tags
