#!/bin/bash

dry_run() {
    sudo certbot certonly \
        --non-interactive \
        --agree-tos \
        --manual \
        -m ${email} \
        --preferred-challenge http \
        --manual-auth-hook /home/mqtt-server/certbot/authenticator.sh \
        -d ${mosquittoDomain} \
        --dry-run
}

run() {
    sudo certbot certonly \
        --non-interactive \
        --agree-tos \
        --manual \
        -m ${email} \
        --preferred-challenge http \
        --manual-auth-hook /home/mqtt-server/certbot/authenticator.sh \
        -d ${mosquittoDomain}
}

update_files() {
    index=${1}
    cd /etc/letsencrypt/archive/${mosquittoDomain}
    sudo cp *${index}* /etc/mosquitto/certs/${mosquittoDomain}/
    cd /etc/mosquitto/certs/${mosquittoDomain}/
    sudo chmod 755 privkey${index}.pem
    echo "Update the mosquitto.conf cert file locations with latest index. Press enter to continue."
    read
    nano /etc/mosquitto/conf.d/mosquitto.conf
    cat /etc/mosquitto/conf.d/mosquitto.conf
    sudo systemctl restart mosquitto
    systemctl status mosquitto
    journalctl -u mosquitto -f
}

start_server() {
    sudo rm -f /tmp/shutdown
    sudo python3 server.py &
    sleep 5
}


mosquittoDomain=${1}
email=${2}
index=${3}
dryRun=${4}
if [[ ${mosquittoDomain} == "" ]]; then
  echo "arg mosquittoDomain must be set"
  exit 1
fi
if [[ ${email} == "" ]]; then
  echo "arg email must be set"
  exit 1
fi
if [[ ${index} == "" ]]; then
  echo "arg index must be set"
  exit 1
fi

echo "open port 80 in the firewall, then press enter to continue"
read

if [[ ${dryRun} == "" ]]; then
    dry_run
    echo
    echo "this was a dry-run"
else
    run
    echo
    echo "cert created, close port 80 in the firewall"
fi
