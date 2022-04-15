FROM alpine

COPY build/pi-sensor-server /app/
COPY server/frontend/build /app/frontend/build

WORKDIR /app

ENTRYPOINT ["/app/pi-sensor-server"]
