#!/bin/bash

sudo apt-get update
sudo apt-get install jq -y

archive_path="/tmp/archive"
install_dir="/home/pi"
mkdir -p ${archive_path}

releaseURL=$(curl -s https://api.github.com/repos/andrewmarklloyd/pi-sensor/releases/latest | jq -r '.assets[] | select(.name | startswith("pi-sensor-v") ) | select(.name | endswith(".tar.gz")) | .browser_download_url')
curl -sL -o /tmp/release.tar.gz "${releaseURL}"
tar xvfz /tmp/release.tar.gz -C "${archive_path}"

mv "${archive_path}/pi-sensor" ${install_dir}
mv "${archive_path}/run.sh" ${install_dir}

# Configure Heroku to get secrets
echo "Enter the Heroku API key to configure the app:"
read -s HEROKU_API_KEY
echo "Enter the name of the sensor source (example: garage, front-door, back-door, etc.)"
read -s SENSOR_SOURCE

tokenCheckError=$(curl -s -n https://api.heroku.com/apps/pi-sensor/config-vars \
  -H "Accept: application/vnd.heroku+json; version=3" \
  -H "Authorization: Bearer ${HEROKU_API_KEY}" | jq -r '.id')
if [[ ${tokenCheckError} != "null" ]]; then
  echo "Unable to authenticate with Heroku, received error '${tokenCheckError}'. Exiting now"
  exit 1
fi

# use ~ as a delimiter
sed -e "s~{{.HEROKU_API_KEY}}~${HEROKU_API_KEY}~" \
    -e "s~{{.SENSOR_SOURCE}}~${SENSOR_SOURCE}~" \
    ${archive_path}/pi-sensor.service.tmpl \
    > ${archive_path}/pi-sensor.service

sudo mv ${archive_path}/pi-sensor.service /etc/systemd/system/
sudo systemctl enable pi-sensor.service
sudo systemctl start pi-sensor.service
rm -rf ${archive_path}

echo "Installation complete"
