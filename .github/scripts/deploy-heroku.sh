#!/bin/bash

set -euo pipefail

if ! command -v jq &> /dev/null; then
  apt-get update
  apt-get install jq -y
fi

app=${1}

get_version() {
  curl -s -X GET \
    -H "api-key: ${PI_APP_DEPLOYER_API_KEY}" \
    https://${app}.herokuapp.com/health
}

heroku container:login
heroku container:push web -a ${app}
heroku container:release web -a ${app}

i=0
while [[ ${version} != ${GITHUB_SHA} ]]; do
  if [[ ${i} -gt 12 ]]; then
    echo "Exceeded max attempts checking deployment health, deployment failed"
    exit 1
  fi
  version=$(get_version)
  i=$((i+1))
  sleep 5
done
