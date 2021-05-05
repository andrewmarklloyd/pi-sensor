#!/bin/bash

build() {
  func=${1}
  cd ${func}
  GOOS=linux GOARCH=arm GOARM=5 go build -o pi-sensor-${func}
}

if [[ ${1} == 'build' ]]; then
  func=${2}
  if [[ ${func} == 'client' ]]; then
    build client || exit 1
  elif [[ ${func} == 'consumer' ]]; then
    build consumer || exit 1
  fi
fi
