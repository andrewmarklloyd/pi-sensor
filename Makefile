.PHONY: build test

# SHELL := /bin/bash

build:
	GOARCH=arm64 GOARM=5 go build -o build/pi-sensor-server server/*.go
	GOOS=linux GOARCH=arm GOARM=5 go build -o build/pi-sensor-agent agent/*.go
	GOOS=linux GOARCH=arm GOARM=5 go build -o build/door-light door-light/*.go

build-frontend:
	./scripts/build-front.sh

build-ci: build build-frontend
	mv build/* .

deploy-dev: build
	scp pi-sensor-agent pi@${IP}:dev-pi-sensor-agent

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf build/
