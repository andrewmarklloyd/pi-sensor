#!/bin/bash


configure_files() {
    mkdir -p /home/mqtt-server/certbot/tmp/.well-known/acme-challenge/

    echo '#!/bin/bash
echo $CERTBOT_VALIDATION > /home/mqtt-server/certbot/tmp/.well-known/acme-challenge/$CERTBOT_TOKEN' > /home/mqtt-server/certbot/authenticator.sh
    chmod +x /home/mqtt-server/certbot/authenticator.sh

    echo '#!/bin/bash
touch /tmp/shutdown' > /home/mqtt-server/certbot/cleanup.sh
    chmod +x /home/mqtt-server/certbot/cleanup.sh

    echo 'from http.server import HTTPServer, SimpleHTTPRequestHandler
import os, threading, time, sys, signal

def shutdown_if_found():
    while True:
        if os.path.isfile("/tmp/shutdown"):
            os.kill(os.getpid(), signal.SIGINT)
        time.sleep(1)

def start_server():
    tmp_dir = os.path.join(os.path.dirname(__file__), "tmp")
    os.chdir(tmp_dir)
    httpd = HTTPServer(("", 80), SimpleHTTPRequestHandler)
    httpd.serve_forever()

print("starting server")
b = threading.Thread(name="background", target=shutdown_if_found)
f = threading.Thread(name="foreground", target=start_server)

b.start()
f.start()' > /home/mqtt-server/certbot/server.py
}

check_ip() {
    ownIP=$(curl -4s ifconfig.me)
    domainIP=$(dig +short ${mosquittoDomain})
    until [ ${ownIP} == ${domainIP} ]; do
        ownIP=$(curl -4s ifconfig.me)
        domainIP=$(dig +short ${mosquittoDomain})
        echo "expected IP ${ownIP} is not ${domainIP}, waiting"
        sleep 10
    done
}

start_certbot() {
    cd /home/mqtt-server/certbot/
    sudo python3 server.py &

    sleep 5

    sudo certbot certonly \
        --non-interactive \
        --agree-tos \
        --manual \
        -m ${email} \
        --preferred-challenge http \
        --manual-auth-hook /home/mqtt-server/certbot/authenticator.sh \
        --manual-cleanup-hook /home/mqtt-server/certbot/cleanup.sh \
        -d ${mosquittoDomain}

    sleep 5
}

config_mosquitto_certs() {
    sudo mkdir -p /etc/mosquitto/certs/${mosquittoDomain}/
    sudo sh -c "cp /etc/letsencrypt/archive/${mosquittoDomain}/* /etc/mosquitto/certs/${mosquittoDomain}/"

    sudo chmod 755 /etc/mosquitto/certs/${mosquittoDomain}/privkey1.pem

    grep cafile /etc/mosquitto/conf.d/mosquitto.conf > /dev/null
    if [[ $? != 0 ]]; then
        sudo sh -c "echo 'cafile /etc/mosquitto/certs/${mosquittoDomain}/chain1.pem
keyfile /etc/mosquitto/certs/${mosquittoDomain}/privkey1.pem
certfile /etc/mosquitto/certs/${mosquittoDomain}/cert1.pem' >> /etc/mosquitto/conf.d/mosquitto.conf"
    fi

    sudo systemctl restart mosquitto
}


mosquittoDomain=${1}
email=${2}

if [[ -z ${mosquittoDomain} ]]; then
    echo "first argument for mosquittoDomain is empty"
    exit 1
fi

if [[ -z ${email} ]]; then
    echo "second argument for email is empty"
    exit 1
fi

configure_files
check_ip
start_certbot
config_mosquitto_certs
