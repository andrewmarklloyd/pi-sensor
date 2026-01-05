#!/bin/bash
echo $CERTBOT_VALIDATION > /home/mqtt-server/certbot/tmp/.well-known/acme-challenge/$CERTBOT_TOKEN
