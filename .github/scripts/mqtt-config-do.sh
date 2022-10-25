#!/bin/bash

set -eu

get_config() {
  vault=${1}
  op item get --vault ${vault} "config" --fields type=concealed --format json
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
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_AGENT_USER}\",\"pattern\":\"sensor/restart\",\"read\":true,\"write\":false}"
}

create_server_user() {
  post user "{\"username\": \"${CLOUDMQTT_SERVER_USER}\",\"password\": \"${CLOUDMQTT_SERVER_PASSWORD}\"}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_SERVER_USER}\",\"pattern\":\"sensor/status\",\"read\":true,\"write\":false}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_SERVER_USER}\",\"pattern\":\"sensor/heartbeat\",\"read\":true,\"write\":false}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_SERVER_USER}\",\"pattern\":\"sensor/restart\",\"read\":false,\"write\":true}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_SERVER_USER}\",\"pattern\":\"ha/#\",\"read\":false,\"write\":true}"
}

create_app_user() {
  post user "{\"username\": \"${CLOUDMQTT_APP_USER}\",\"password\": \"${CLOUDMQTT_APP_PASSWORD}\"}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_APP_USER}\",\"pattern\":\"sensor/status\",\"read\":true,\"write\":false}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_APP_USER}\",\"pattern\":\"sensor/heartbeat\",\"read\":false,\"write\":true}"
}

create_ha_user() {
  post user "{\"username\": \"${CLOUDMQTT_HA_USER}\",\"password\": \"${CLOUDMQTT_HA_PASSWORD}\"}"
  post acl "{\"type\":\"topic\",\"username\":\"${CLOUDMQTT_HA_USER}\",\"pattern\":\"ha/#\",\"read\":true,\"write\":false}"
}

# agent config
config=$(get_config pi-sensor-agent)
fields="CLOUDMQTT_URL
CLOUDMQTT_AGENT_USER
CLOUDMQTT_AGENT_PASSWORD"
for f in ${fields}; do
  export ${f}=$(echo ${config} | jq -r ".[] | select(.label==\"${f}\").value")
done

# server config
config=$(get_config pi-sensor-server)
fields="CLOUDMQTT_URL
CLOUDMQTT_SERVER_USER
CLOUDMQTT_SERVER_PASSWORD
CLOUDMQTT_APIKEY
CLOUDMQTT_HA_USER
CLOUDMQTT_HA_PASSWORD"
for f in ${fields}; do
  export ${f}=$(echo ${config} | jq -r ".[] | select(.label==\"${f}\").value")
done


create_agent_user
create_server_user
create_ha_user
