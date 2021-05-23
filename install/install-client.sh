#!/bin/bash

sudo apt-get update
sudo apt-get install jq -y

archive_path="/tmp/pi-sensor"
install_dir="/home/pi"
mkdir -p ${archive_path}

latestVersion=$(curl --silent "https://api.github.com/repos/andrewmarklloyd/pi-sensor/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
curl -sL https://github.com/andrewmarklloyd/pi-sensor/archive/${latestVersion}.tar.gz | tar xvfz - -C "${archive_path}" --strip 1 > /dev/null

binaryUrl=$(curl -s https://api.github.com/repos/andrewmarklloyd/pi-sensor/releases/latest | jq -r '.assets[] | select(.name == "pi-sensor") | .browser_download_url')
curl -sL ${binaryUrl} -o ${archive_path}/pi-sensor
chmod +x ${archive_path}/pi-sensor
mv ${archive_path}/pi-sensor ${install_dir}/

echo "Enter the CloudMQTT URL:"
read -r CLOUDMQTT_URL
echo "Enter the CloudMQTT topic to publish messages:"
read -s TOPIC
echo "Enter the name of the sensor (example front-door, garage, etc.):"
read -s SENSOR_SOURCE

# use ~ as a delimiter
sed -e "s~{{.CLOUDMQTT_URL}}~${CLOUDMQTT_URL}~" \
    -e "s~{{.TOPIC}}~${TOPIC}~" \
    -e "s~{{.SENSOR_SOURCE}}~${SENSOR_SOURCE}~" \
    ${archive_path}/install/pi-sensor.service.tmpl \
    > ${archive_path}/install/pi-sensor.service

sudo mv ${archive_path}/install/pi-sensor.service /etc/systemd/system/
sudo systemctl enable pi-sensor.service
sudo systemctl start pi-sensor.service
rm -rf ${archive_path}

echo "Installation complete"
