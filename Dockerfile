FROM golang:1.16.15 as builder

WORKDIR /app
COPY . /app
RUN make vet
RUN make test
RUN make build-ci

FROM alpine
COPY --from=builder /app/build/pi-sensor-server /app/
COPY --from=builder /app/frontend/build /app/frontend/build

WORKDIR /app

ENTRYPOINT ["/app/pi-sensor-server"]
