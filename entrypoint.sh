#!/bin/sh


if [ ${RUNTIME} == "D_O" ]; then
    /app/do-app-firewall-entrypoint
fi

unset DO_ACCESS_TOKEN
unset DO_FIREWALL_ID

exp=$(op read op://pi-sensor-server/config/OP_TOKEN_EXP)
remaining=$(( ($(date -d ${exp} +%s) - $(date +%s) )/(60*60*24) ))
echo "Token remaining days: ${remaining}"
/app/op run --env-file="/app/.env.server.tmpl" -- /app/pi-sensor-server
