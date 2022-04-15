FROM alpine

COPY build/pi-sensor-server /app/
COPY server/frontend/build /app/frontend/build

ENTRYPOINT ["/app/pi-sensor-server"]
