.PHONY: build test

build:
	GOARCH=arm64 GOARM=5 go build -o build/pi-sensor-server server/*.go
	GOOS=linux GOARCH=arm GOARM=5 go build -o build/pi-sensor-agent agent/*.go
	GOOS=linux GOARCH=arm GOARM=5 go build -o build/door-light scripts/door-light/main.go

build-frontend:
	curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.37.2/install.sh | bash
	export NVM_DIR="$HOME/.nvm"
	cd server/frontend
	nvm install
	nvm use
	npm install
	npm run build

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
