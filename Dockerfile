FROM golang:1.19-alpine as builder
RUN go install github.com/andrewmarklloyd/do-app-firewall-entrypoint@latest

RUN apk add curl
RUN curl -sSfo op.zip \
  https://cache.agilebits.com/dist/1P/op2/pkg/v2.7.1/op_linux_amd64_v2.7.1.zip \
  && unzip -od /usr/local/bin/ op.zip \
  && rm op.zip

FROM alpine
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
ADD ./opconnect.crt /usr/local/share/ca-certificates/opconnect.crt
RUN update-ca-certificates

COPY --from=builder /go/bin/do-app-firewall-entrypoint /app/do-app-firewall-entrypoint
COPY --from=builder /usr/local/bin/op /app/op
COPY build/pi-sensor-server /app/
COPY frontend/build /app/frontend/build
COPY entrypoint.sh /entrypoint.sh
COPY .env.server.tmpl /app/.env.server.tmpl
WORKDIR /app

ENTRYPOINT ["/entrypoint.sh"]
