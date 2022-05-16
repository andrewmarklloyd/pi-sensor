#!/bin/bash

set -eu

get_config() {
  curl -s -n https://api.heroku.com/apps/${HEROKU_APP}/config-vars \
    -H "Accept: application/vnd.heroku+json; version=3" \
    -H "Authorization: Bearer ${HEROKU_API_KEY}"
}

post() {
  endpoint=${1}
  payload=${2}
  curl -s -XPOST -u :${CLOUDMQTT_APIKEY} \
    -d "${payload}" \
    -H "Content-Type:application/json" https://api.cloudmqtt.com/api/${endpoint}
}

create_agent_user() {
  post user "{\"username\": \"${CLOUDMQTT_AGENT_USER}\",\"password\": \"${CLOUDMQTT_AGENT_PASSWORD}\"}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_AGENT_USER}\",\"pattern\":\"sensor/status\",\"read\":false,\"write\":true}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_AGENT_USER}\",\"pattern\":\"sensor/heartbeat\",\"read\":false,\"write\":true}"
}

create_server_user() {
  post user "{\"username\": \"${CLOUDMQTT_SERVER_USER}\",\"password\": \"${CLOUDMQTT_SERVER_PASSWORD}\"}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_SERVER_USER}\",\"pattern\":\"sensor/status\",\"read\":true,\"write\":false}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_SERVER_USER}\",\"pattern\":\"sensor/heartbeat\",\"read\":true,\"write\":false}"
}

config=$(get_config)
CLOUDMQTT_APIKEY=$(echo ${config} | jq -r '.CLOUDMQTT_APIKEY')
CLOUDMQTT_AGENT_USER=$(echo ${config} | jq -r '.CLOUDMQTT_AGENT_USER')
CLOUDMQTT_AGENT_PASSWORD=$(echo ${config} | jq -r '.CLOUDMQTT_AGENT_PASSWORD')
CLOUDMQTT_SERVER_USER=$(echo ${config} | jq -r '.CLOUDMQTT_SERVER_USER')
CLOUDMQTT_SERVER_PASSWORD=$(echo ${config} | jq -r '.CLOUDMQTT_SERVER_PASSWORD')


create_agent_user >/dev/null
create_server_user >/dev/null
