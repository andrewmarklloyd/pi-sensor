#!/bin/bash


if [[ -z ${HEROKU_API_KEY} ]]; then
  echo "HEROKU_API_KEY env var not set, exiting now"
  exit 1
fi

vars=$(curl -s -n https://api.heroku.com/apps/pi-sensor/config-vars \
  -H "Accept: application/vnd.heroku+json; version=3" \
  -H "Authorization: Bearer ${HEROKU_API_KEY}")

export CLOUDMQTT_URL=$(echo $vars | jq -r '.CLOUDMQTT_URL')
unset HEROKU_API_KEY

/home/pi/pi-sensor
