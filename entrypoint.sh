#!/bin/sh


if [ ${RUNTIME} == "D_O" ]; then
    /app/do-app-firewall-entrypoint
fi

unset DO_TOKEN
unset DO_FIREWALL_ID

# /app/op run --env-file="./.env.tmpl" -- /app/pi-sensor-server
/app/pi-sensor-server
