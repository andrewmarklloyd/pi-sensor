ARG GO_VERSION=latest
FROM golang:${GO_VERSION}-alpine as builder
RUN go install github.com/andrewmarklloyd/do-app-firewall-entrypoint@v0.1.0

RUN apk add curl
ENV OP_VERSION=v2.26.1
RUN curl -sSfo op.zip \
  https://cache.agilebits.com/dist/1P/op2/pkg/${OP_VERSION}/op_linux_amd64_${OP_VERSION}.zip \
  && unzip -od /usr/local/bin/ op.zip \
  && rm op.zip

FROM alpine

COPY --from=builder /go/bin/do-app-firewall-entrypoint /app/do-app-firewall-entrypoint
COPY --from=builder /usr/local/bin/op /usr/local/bin/op
COPY build/pi-sensor-server /app/
RUN chmod +x /app/pi-sensor-server
COPY build/op-limit-check-entry /app/
RUN chmod +x /app/op-limit-check-entry
COPY frontend/dist /app/frontend/dist
COPY entrypoint.sh /entrypoint.sh
COPY .env.server.tmpl /app/.env.server.tmpl
WORKDIR /app

ENTRYPOINT ["/entrypoint.sh"]