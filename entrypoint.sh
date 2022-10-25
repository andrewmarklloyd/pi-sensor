#!/bin/sh


if [ ${RUNTIME} == "D_O" ]; then
    export DO_ACCESS_TOKEN=$(op read op://pi-sensor-server/config/DO_TOKEN)
    /app/do-app-firewall-entrypoint
    unset DO_ACCESS_TOKEN
    unset DO_FIREWALL_ID
fi

/app/op run --env-file="/app/.env.server.tmpl" -- /app/pi-sensor-server
