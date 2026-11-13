FROM alpine:3.23 AS qlty

ARG TARGETARCH
RUN apk add --no-cache curl xz \
    && case "$TARGETARCH" in \
         amd64) qlty_target=x86_64-unknown-linux-musl; qlty_sha=a275a616f5362985f5c316fd4676480cb73ae9e7a75d58f29529ce953bbfa134 ;; \
         arm64) qlty_target=aarch64-unknown-linux-musl; qlty_sha=854cce3e5987e143abe04b9500682e8edf07a112c07bcf47d5d48bbdb2518da7 ;; \
         *) exit 1 ;; \
       esac \
    && curl --fail --location --retry 3 "https://github.com/qltysh/qlty/releases/download/v0.668.0/qlty-${qlty_target}.tar.xz" --output /tmp/qlty.tar.xz \
    && echo "$qlty_sha  /tmp/qlty.tar.xz" | sha256sum -c - \
    && mkdir /opt/qlty \
    && tar -xJf /tmp/qlty.tar.xz -C /opt/qlty --strip-components=1

FROM golang:alpine

RUN apk add --no-cache git \
    && addgroup -S devpulse \
    && adduser -S -G devpulse -h /home/devpulse devpulse

COPY --from=qlty /opt/qlty/qlty /usr/local/bin/qlty
COPY --from=qlty /opt/qlty/LICENSE.md /opt/qlty/LICENSE.md

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080 50051

USER devpulse

CMD ["go", "run", "."]

