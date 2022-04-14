FROM golang:1.16 as builder

WORKDIR /app
COPY . .

ENV GO111MODULE=on

FROM scratch

WORKDIR /app

COPY --from=builder /app/build/pi-sensor-server /usr/bin/
RUN mkdir /usr/bin/frontend
COPY server/frontend/build frontend/

ENTRYPOINT ["/usr/bin/pi-sensor-server"]
