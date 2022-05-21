#!/bin/bash

set -eu

aws s3api put-bucket-versioning --bucket ${BUCKETEER_BUCKET_NAME} --versioning-configuration Status=Enabled
