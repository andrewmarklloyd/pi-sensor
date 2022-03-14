#!/bin/bash


response=$(curl -s -X POST -H "api-key: ${PI_APP_DEPLOYER_API_KEY}" -d "{\"repoName\":\"andrewmarklloyd/pi-sensor\",\"manifestName\":\"door-light\",\"action\":\"${ACTION}\"}" https://pi-app-deployer.herokuapp.com/service)
echo ${response}
if [[ $(echo ${response} | jq -r '.status') != "success" ]]; then
  exit 1
fi
