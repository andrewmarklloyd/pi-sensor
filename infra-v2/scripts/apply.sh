#!/bin/bash

set -u

skipAnsible="${1:-}"

RELATIVE_SCRIPT_DIR="${(%):-%N}"
if [[ $? != 0 ]]; then
  RELATIVE_SCRIPT_DIR=${BASH_SOURCE[0]}
fi

SCRIPT_DIR=$( cd -- "$( dirname -- "${RELATIVE_SCRIPT_DIR}" )" &> /dev/null && pwd )

source ${SCRIPT_DIR}/setup.sh
cd ${SCRIPT_DIR}/../terraform
terraform apply
if [[ $? != 0 ]]; then
  echo "terraform apply failed, not continuing"
  exit 1
fi

ip=$(terraform output -raw ip_address)
cd ${SCRIPT_DIR}/../
check_ssh ${ip}

if [[ ! -z ${skipAnsible} && ${skipAnsible} == "skip-ansible" ]]; then
  exit 0
fi

echo ${ip} | tee /tmp/hosts

ansible-playbook -i /tmp/hosts ansible/playbook.yaml
