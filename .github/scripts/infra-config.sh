#!/bin/bash

set -euo pipefail

get_config() {
  curl -s -n https://api.heroku.com/apps/${HEROKU_APP}/config-vars \
    -H "Accept: application/vnd.heroku+json; version=3" \
    -H "Authorization: Bearer ${HEROKU_API_KEY}"
}

aws_bucket_config() {
    aws s3api put-bucket-versioning --bucket ${BUCKETEER_BUCKET_NAME} --versioning-configuration Status=Enabled
    aws s3api get-bucket-versioning --bucket ${BUCKETEER_BUCKET_NAME}
    aws s3api put-bucket-lifecycle-configuration \
        --bucket ${BUCKETEER_BUCKET_NAME} \
        --lifecycle-configuration file://.github/scripts/assets/lifecycle.json
    aws s3api get-bucket-lifecycle-configuration --bucket ${BUCKETEER_BUCKET_NAME}
}


config=$(get_config)
export AWS_ACCESS_KEY_ID=$(echo ${config} | jq -r '.BUCKETEER_AWS_ACCESS_KEY_ID')
export AWS_REGION=$(echo ${config} | jq -r '.BUCKETEER_AWS_REGION')
export AWS_SECRET_ACCESS_KEY=$(echo ${config} | jq -r '.BUCKETEER_AWS_SECRET_ACCESS_KEY')
export BUCKETEER_BUCKET_NAME=$(echo ${config} | jq -r '.BUCKETEER_BUCKET_NAME')

aws_bucket_config
