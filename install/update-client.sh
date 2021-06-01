#!/bin/bash

forceUpdate=${1}

latestRelease=$(curl -s https://api.github.com/repos/andrewmarklloyd/pi-sensor/releases/latest | jq -r .tag_name)
currentRelease=$(cat /home/pi/.currentRelease)
if [[ -z ${currentRelease} ]]; then
    echo ${latestRelease} > /home/pi/.currentRelease
fi

if [[ ${latestRelease} != ${currentRelease} || ${forceUpdate} == "-force" ]]; then
    echo "New version available: ${latestRelease}, downloading now"
    archive_path="/tmp/archive"
    install_dir="/home/pi"
    mkdir -p ${archive_path}

    releaseURL=$(curl -s https://api.github.com/repos/andrewmarklloyd/pi-sensor/releases/latest | jq -r ".assets[] | select(.name | startswith(\"pi-sensor-${latestRelease}\") ) | select(.name | endswith(\".tar.gz\")) | .browser_download_url")
    curl -sL -o /tmp/release.tar.gz "${releaseURL}"
    tar xvfz /tmp/release.tar.gz -C "${archive_path}"

    sudo systemctl stop pi-sensor.service
    mv "${archive_path}/pi-sensor" /home/pi/
    sudo systemctl start pi-sensor.service
    rm -rf ${archive_path}
    rm /tmp/release.tar.gz
    echo "Update complete"
fi
