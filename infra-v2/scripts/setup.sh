#!/bin/bash

set -u

RELATIVE_SCRIPT_DIR="${(%):-%N}"
if [[ $? != 0 ]]; then
  RELATIVE_SCRIPT_DIR=${BASH_SOURCE[0]}
fi

SCRIPT_DIR=$( cd -- "$( dirname -- "${RELATIVE_SCRIPT_DIR}" )" &> /dev/null && pwd )

export TF_VAR_ssh_inbound_ip=$(curl -4s ifconfig.me)
eval $(op inject -i ${SCRIPT_DIR}/.op.tmpl)

cd ${SCRIPT_DIR}/../terraform/
tfenv install
tfenv use
terraform init

check_ssh() {
  ip=${1}
  success='false'
  echo "Checking for ssh access"
  until [ ${success} == 'true' ]; do
    ssh pi-sensor-data@${ip} exit
    code=$(echo $?)
    if [[ ${code} == 0 ]]; then
      success='true'
    else
      echo "exit code: ${code}"
      sleep 15
    fi
  done
}

check_docker() {
  ip=${1}
  success='false'
  echo "Checking for docker running"
  until [ ${success} == 'true' ]; do
    ssh pi-sensor-data@${ip} docker ps
    code=$(echo $?)
    if [[ ${code} == 0 ]]; then
      success='true'
    else
      echo "exit code: ${code}"
      sleep 5
    fi
  done
}
