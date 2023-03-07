.PHONY: build test

GIT_REV=`git rev-parse --short HEAD`
GIT_TREE_STATE=$(shell (git status --porcelain | grep -q .) && echo $(GIT_REV)-dirty || echo $(GIT_REV))

build:
	CGO_ENABLED=0 GOARCH=amd64 go build -ldflags="-X 'github.com/andrewmarklloyd/pi-sensor/cmd.version=$(GIT_TREE_STATE)'" -o build/pi-sensor-server main.go
	GOOS=linux GOARCH=arm GOARM=5 go build -ldflags="-X 'main.version=$(GIT_TREE_STATE)'" -o build/pi-sensor-agent agent/main.go
	GOOS=linux GOARCH=arm GOARM=5 go build -o build/agent-log-forwarder log-forwarder/main.go


build-frontend: export REACT_APP_VAPID_PUBLIC_KEY := $(shell echo ${VAPID_PUBLIC_KEY})
build-frontend:
	REACT_APP_VERSION=$(GIT_TREE_STATE) ./.github/scripts/build-front.sh

build-ci: build build-frontend
	cp ./build/* .
	git diff --exit-code ':!frontend/package-lock.json'

build-dev:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.version=`git rev-parse HEAD`'" -o build/pi-sensor-server server/*.go

deploy-dev: build
	scp pi-sensor-agent pi@${IP}:dev-pi-sensor-agent

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf build/
	rm -rf frontend/build/
