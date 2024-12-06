#!/bin/sh

/app/do-app-firewall-entrypoint

unset DO_ACCESS_TOKEN
unset FIREWALL_NAME
unset STATIC_INBOUND_IPS
unset FIREWALL_PORT

/app/op-limit-check-entry

/app/tailscaled --tun=userspace-networking &
until /app/tailscale up --authkey=${TAILSCALE_AUTHKEY}
do
    sleep 1
done
echo Tailscale started

unset TAILSCALE_AUTHKEY
op run --env-file="/app/.env.server.tmpl" -- /app/pi-sensor-server
