FROM golang:1.19-alpine as builder
RUN go install github.com/andrewmarklloyd/do-app-firewall-entrypoint@latest

FROM ubuntu:22.04 as opbuilder
RUN apt update && apt install curl gpg -y
RUN curl -sS https://downloads.1password.com/linux/keys/1password.asc | \
    gpg --dearmor --output /usr/share/keyrings/1password-archive-keyring.gpg
RUN echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/1password-archive-keyring.gpg] https://downloads.1password.com/linux/debian/$(dpkg --print-architecture) stable main" | tee /etc/apt/sources.list.d/1password.list
RUN mkdir -p /etc/debsig/policies/AC2D62742012EA22/
RUN curl -sS https://downloads.1password.com/linux/debian/debsig/1password.pol | tee /etc/debsig/policies/AC2D62742012EA22/1password.pol
RUN mkdir -p /usr/share/debsig/keyrings/AC2D62742012EA22
RUN curl -sS https://downloads.1password.com/linux/keys/1password.asc | gpg --dearmor --output /usr/share/debsig/keyrings/AC2D62742012EA22/debsig.gpg
RUN apt update && apt install 1password-cli
RUN which op

FROM alpine
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
ADD ./opconnect.crt /usr/local/share/ca-certificates/opconnect.crt
RUN update-ca-certificates

COPY --from=builder /go/bin/do-app-firewall-entrypoint /app/do-app-firewall-entrypoint
COPY --from=opbuilder /usr/bin/op /app/op
COPY build/pi-sensor-server /app/
COPY frontend/build /app/frontend/build
COPY entrypoint.sh /entrypoint.sh
COPY .env.tmpl /app/.env.tmpl
WORKDIR /app

ENTRYPOINT ["/entrypoint.sh"]
