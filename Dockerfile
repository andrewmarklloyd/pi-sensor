FROM golang:1.22-alpine as builder
RUN go install github.com/andrewmarklloyd/do-app-firewall-entrypoint@latest

RUN apk add curl
ENV OP_VERSION=v2.22.0
RUN curl -sSfo op.zip \
  https://cache.agilebits.com/dist/1P/op2/pkg/${OP_VERSION}/op_linux_amd64_${OP_VERSION}.zip \
  && unzip -od /usr/local/bin/ op.zip \
  && rm op.zip

FROM alpine

COPY --from=builder /go/bin/do-app-firewall-entrypoint /app/do-app-firewall-entrypoint
COPY --from=builder /usr/local/bin/op /app/op
COPY build/pi-sensor-server /app/
COPY frontend/build /app/frontend/build
COPY entrypoint.sh /entrypoint.sh
COPY .env.server.tmpl /app/.env.server.tmpl
WORKDIR /app

ENTRYPOINT ["/entrypoint.sh"]
