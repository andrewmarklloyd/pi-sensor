FROM golang:1.16 as builder

WORKDIR /app
COPY . .

ENV GO111MODULE=on

RUN make build
RUN make build-frontend

FROM alpine

WORKDIR /app
COPY --from=builder /app/build/pi-sensor-server /app/
COPY --from=builder /app/server/frontend/build /app/frontend/build

ENTRYPOINT ["/app/pi-sensor-server"]
