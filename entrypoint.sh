#!/bin/sh


if [ ${RUNTIME} == "D_O" ]; then
    /app/do-app-firewall-entrypoint
fi

unset DO_TOKEN
unset DO_FIREWALL_ID

/app/pi-sensor-server
