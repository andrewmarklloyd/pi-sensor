FROM golang:1.19-alpine as builder

RUN apk add curl
RUN curl -sSfo op.zip \
  https://cache.agilebits.com/dist/1P/op2/pkg/v2.18.0-beta.01/op_linux_amd64_v2.18.0-beta.01.zip \
  && unzip -od /usr/local/bin/ op.zip \
  && rm op.zip

FROM alpine

COPY --from=builder /usr/local/bin/op /app/op
COPY build/pi-sensor-server /app/
COPY frontend/build /app/frontend/build
COPY entrypoint.sh /entrypoint.sh
COPY .env.server.tmpl /app/.env.server.tmpl
WORKDIR /app

ENTRYPOINT ["/entrypoint.sh"]
