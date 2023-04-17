#!/bin/sh


/app/op run --env-file="/app/.env.server.tmpl" -- /app/pi-sensor-server
