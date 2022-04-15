#!/bin/bash

set -euo pipefail

if ! command -v jq &> /dev/null; then
  apt-get update
  apt-get install jq -y
fi

app=${1}

get_version() {
  curl -s -X GET \
    -H "api-key: ${SERVER_API_KEY}" \
    https://${app}.herokuapp.com/health | jq -r '.version'
}

heroku container:login
heroku container:push web -a ${app}
heroku container:release web -a ${app}

echo "Deploying version ${GITHUB_SHA}"

version="unknown"
i=0
while [[ ${version} != ${GITHUB_SHA} ]]; do
  echo "Attempt number ${i}, deployed version: ${version}"
  if [[ ${i} -gt 12 ]]; then
    echo "Exceeded max attempts checking deployment health, deployment failed"
    exit 1
  fi
  version=$(get_version)
  i=$((i+1))
  sleep 5
done
