#!/bin/bash


docker-compose up -d

export REDIS_TLS_URL=redis://127.0.0.1:6379
export REDIS_URL=redis://127.0.0.1:6379
export DATABASE_URL=postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable
export CLOUDMQTT_URL=mqtt://user:pass@127.0.0.1:1883
export CLOUDMQTT_SERVER_PASSWORD=abc
export CLOUDMQTT_SERVER_USER=abc
export PORT=8080

export GOOGLE_CLIENT_ID=$(op read op://pi-sensor-server/config/GOOGLE_CLIENT_ID)
export GOOGLE_CLIENT_SECRET=$(op read op://pi-sensor-server/config/GOOGLE_CLIENT_SECRET)
export REDIRECT_URL=http://localhost:8080/google/callback
export AUTHORIZED_USERS=$(op read op://pi-sensor-server/config/AUTHORIZED_USERS)

make build-frontend
go run main.go
