.PHONY: build test

build:
	CGO_ENABLED=0 GOARCH=amd64 go build -ldflags="-X 'main.version=`git rev-parse HEAD`'" -o build/pi-sensor-server server/*.go
	GOOS=linux GOARCH=arm GOARM=5 go build -o build/pi-sensor-agent agent/*.go
	GOOS=linux GOARCH=arm GOARM=5 go build -o build/door-light door_light/*.go

build-frontend:
	./.github/scripts/build-front.sh

deploy-dev: build
	scp pi-sensor-agent pi@${IP}:dev-pi-sensor-agent

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf build/
