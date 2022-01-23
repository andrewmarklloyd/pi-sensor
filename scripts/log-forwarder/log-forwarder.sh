#!/bin/bash

LOG_SERVER_API_TOKEN=''
LOG_SERVER_URL='http://localhost:8080/api/logs/submit'

while IFS= read -r line; do
  curl -H "log-server-api-token: ${LOG_SERVER_API_TOKEN}" -X POST -d "${line}" "${LOG_SERVER_URL}"
done
