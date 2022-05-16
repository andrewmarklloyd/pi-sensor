#!/bin/bash


set -euo pipefail

if ! command -v jq &> /dev/null; then
  apt-get update
  apt-get install jq -y
fi

if ! command -v heroku &> /dev/null; then
  curl https://cli-assets.heroku.com/install-ubuntu.sh | sh
fi

deploy() {
  echo "Deploying version ${SHORT_SHA}"
  heroku container:login
  heroku container:push web -a ${app}
  heroku container:release web -a ${app}
  health_check
}

get_version() {
  curl -s -X GET \
    -H "api-key: ${SERVER_API_KEY}" \
    https://${app}.herokuapp.com/health | jq -r '.version'
}

health_check() {
  version=$(get_version)
  i=0
  echo "Waiting for deployed version to be: ${SHORT_SHA}"
  while [[ ${version} != ${SHORT_SHA} ]]; do
    echo "Attempt number ${i}, deployed version: ${version}"
    if [[ ${i} -gt 12 ]]; then
      echo "Exceeded max attempts checking deployment health, deployment failed"
      exit 1
    fi
    version=$(get_version)
    i=$((i+1))
    sleep 5
  done

  echo "Successfully deployed version ${version}"
  exit 0
}

SHORT_SHA=$(echo ${GITHUB_SHA} | cut -c1-7)
app=${1}
deploy
