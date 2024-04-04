#!/bin/sh

/app/do-app-firewall-entrypoint

unset DO_ACCESS_TOKEN
unset DO_FIREWALL_ID

/app/op run --env-file="/app/.env.server.tmpl" -- /app/pi-sensor-server
