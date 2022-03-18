.PHONY: build

build:
	GOARCH=arm64 GOARM=5 go build -o pi-sensor-server server/*.go
	GOOS=linux GOARCH=arm GOARM=5 go build -o pi-sensor-agent agent/*.go
	GOOS=linux GOARCH=arm GOARM=5 go build -o door-light scripts/door-light/main.go

deploy-dev: build
	scp pi-sensor-agent pi@${IP}:dev-pi-sensor-agent
