FROM golang:1.16 as builder

WORKDIR /app
COPY . .

ENV GO111MODULE=on

RUN make build
# TODO: add front end to dockerfile

FROM scratch

WORKDIR /app

COPY --from=builder /app/build/pi-sensor-server /usr/bin/

ENTRYPOINT ["/usr/bin/pi-sensor-server"]
