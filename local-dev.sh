#!/bin/bash


docker-compose up -d

export REDIS_URL=redis://127.0.0.1:6379
export DATABASE_URL=postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable
export CLOUDMQTT_URL=mqtt://user:pass@127.0.0.1:1883
export CLOUDMQTT_SERVER_PASSWORD=abc
export CLOUDMQTT_SERVER_USER=abc
export CLOUDMQTT_AGENT_USER=abc
export CLOUDMQTT_AGENT_PASSWORD=abc
export MOCK_MODE=true
export PORT=8080
# random uuid
export ENCRYPTION_KEY=074d3351-4c59-434a-ba52-4bedf972
export GOOGLE_CLIENT_ID=$(op read op://pi-sensor-server/config/GOOGLE_CLIENT_ID)
export GOOGLE_CLIENT_SECRET=$(op read op://pi-sensor-server/config/GOOGLE_CLIENT_SECRET)
export REDIRECT_URL=http://localhost:8080/google/callback
export AUTHORIZED_USERS=$(op read op://pi-sensor-server/config/AUTHORIZED_USERS)
export VAPID_PUBLIC_KEY=$(op read op://pi-sensor-server/config/VAPID_PUBLIC_KEY)
export REACT_APP_VAPID_PUBLIC_KEY=${VAPID_PUBLIC_KEY}
export VAPID_PRIVATE_KEY=$(op read op://pi-sensor-server/config/VAPID_PRIVATE_KEY)


if [[ ${1} != source ]]; then
    cd frontend/
    npm install
    npm run build
    cd ../
    go run main.go
fi
