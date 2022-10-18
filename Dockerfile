FROM alpine

RUN apk add curl jq

COPY build/pi-sensor-server /app/
COPY frontend/build /app/frontend/build
COPY entrypoint.sh /entrypoint.sh
WORKDIR /app

ENTRYPOINT ["/entrypoint.sh"]
