#!/bin/bash


docker compose up -d

export REDIS_URL=redis://127.0.0.1:6379
export DATABASE_URL=postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable
export MOSQUITTO_DOMAIN=127.0.0.1
export MOSQUITTO_SERVER_PASSWORD=abc
export MOSQUITTO_SERVER_USER=abc
export MOSQUITTO_AGENT_USER=abc
export MOSQUITTO_AGENT_PASSWORD=abc
export MOSQUITTO_PROTOCOL=mqtt
export MOCK_MODE=true
export PORT=8080
# random uuid
export ENCRYPTION_KEY=074d3351-4c59-434a-ba52-4bedf972
export GOOGLE_CLIENT_ID=$(op read op://pi-sensor-server/config/GOOGLE_CLIENT_ID)
export GOOGLE_CLIENT_SECRET=$(op read op://pi-sensor-server/config/GOOGLE_CLIENT_SECRET)
export REDIRECT_URL=http://localhost:8080/google/callback
export AUTHORIZED_USERS=$(op read op://pi-sensor-server/config/AUTHORIZED_USERS)
# random uuid
export SESSION_SECRET=0c433325-afeb-4a84-85aa-a88edc069d00

if [[ ${1} != source ]]; then
    cd frontend/
    npm install
    npm run build
    cd ../
    go run main.go
fi
