FROM golang:1.19-alpine as builder
RUN go install github.com/andrewmarklloyd/do-app-firewall-entrypoint@latest

FROM alpine

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
ADD ./opconnect.crt /usr/local/share/ca-certificates/opconnect.crt
RUN update-ca-certificates

COPY --from=builder /go/bin/do-app-firewall-entrypoint /app/do-app-firewall-entrypoint
COPY build/pi-sensor-server /app/
COPY frontend/build /app/frontend/build
COPY entrypoint.sh /entrypoint.sh
WORKDIR /app

ENTRYPOINT ["/entrypoint.sh"]
