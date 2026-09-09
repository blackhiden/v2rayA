FROM --platform=$BUILDPLATFORM node:22-alpine AS builder-web
WORKDIR /app
COPY ./gui/package.json ./gui/yarn.lock ./
RUN yarn install
COPY ./gui .
RUN echo "network-timeout 600000" >> .yarnrc
RUN yarn config set registry https://registry.npm.taobao.org
RUN yarn config set sass_binary_site https://cdn.npm.taobao.org/dist/node-sass -g
RUN yarn cache clean && yarn && yarn build

FROM --platform=$BUILDPLATFORM golang:trixie AS builder
WORKDIR /app
# Define the build arguments passed from GitHub Actions
ARG V2RAYA_VERSION=v0.0.0
ARG V2RAYA_COMMIT=unknown
ARG BUILD_DATE=unknown
RUN set -eux;     \
    apt update -y; \
    apt install -y --no-install-recommends       \
        ca-certificates       \
        build-essential;     \
    apt-mark showmanual > /savedAptMark.txt
RUN set -eux;   \
    apt-mark auto '.*' > /dev/null ;	\
    apt-mark manual $(cat /savedAptMark.txt) > /dev/null; 	\
    apt purge -y --auto-remove -o APT::AutoRemove::RecommendsImportant=false;     \
    apt clean;     \
    apt autoclean;     \
    rm -rf /var/lib/apt/lists/*

# built v2rayaA
COPY ./service/go.mod ./service/go.sum ./
COPY ./core/xray ./xray/
RUN set -eux;   \
    go mod tidy; \
    go mod download
COPY ./service .
RUN go mod tidy
COPY --from=builder-web /web ./server/router/web
RUN set -eux;   \
    CGO_ENABLED=1 \
        GOOS=linux \
        go build \
            -buildvcs=false \
            -ldflags="-s -w -X 'main.Version=${V2RAYA_VERSION}' -X 'main.Commit=${V2RAYA_COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" \
            -o ./v2raya . ;  \
    chmod +x ./v2raya

FROM --platform=$BUILDPLATFORM golang:trixie AS builder_core
WORKDIR /app
# Define the build arguments passed from GitHub Actions
ARG XCORE_VERSION=v0.0.0
ARG XCORE_COMMIT=unknown
ARG BUILD_DATE=unknown
RUN set -eux;     \
    apt update -y; \
    apt install -y --no-install-recommends       \
        ca-certificates       \
        build-essential;     \
    apt-mark showmanual > /savedAptMark.txt
RUN set -eux;   \
    apt-mark auto '.*' > /dev/null ;	\
    apt-mark manual $(cat /savedAptMark.txt) > /dev/null; 	\
    apt purge -y --auto-remove -o APT::AutoRemove::RecommendsImportant=false;     \
    apt clean;     \
    apt autoclean;     \
    rm -rf /var/lib/apt/lists/*
# built v2rayA_core
COPY ./core/go.mod ./core/go.sum ./
COPY ./core/xray ./xray/
RUN set -eux;   \
    go mod tidy; \
    go mod download
COPY ./core .
RUN go mod tidy
COPY --from=builder-web /web ./server/router/web
RUN set -eux;   \
    export BUILD_DATE="$(date +%Y-%m-%d)";   \
    CGO_ENABLED=1 \
        GOOS=linux \
        go build \
            -buildvcs=false \
            -ldflags="-s -w -X 'main.Version=${XCORE_VERSION}' -X 'main.Commit=${XCORE_COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" \
            -o ./v2raya_core ./main ;  \
    chmod +x ./v2raya_core

FROM debian:trixie-slim
ENV TZ = "Asia/Jakarta"
SHELL ["/bin/bash", "-c"]
RUN set -eux; 	\
    [ ! -f /etc/localtime ] && ln -s /usr/share/zoneinfo/$TZ /etc/localtime; 	\
    echo $TZ > /etc/timezone; 	\
    apt-get update
WORKDIR /app
RUN set -eux; 	\
    apt update; \
    apt install -y --no-install-recommends \
        ca-certificates \
        iptables \
        iproute2; \
    apt-mark showmanual > /savedAptMark.txt
RUN set -eux;   \
    apt-mark auto '.*' > /dev/null ;	\
    apt-mark manual $(cat /savedAptMark.txt) > /dev/null; 	\
    apt-get purge -y --auto-remove -o APT::AutoRemove::RecommendsImportant=false;     \
    apt-get clean;     \
    apt-get autoclean;     \
    rm -rf /var/lib/apt/lists/*
# Copy binaries into system PATH directories
COPY --from=builder /app/v2raya /usr/bin/v2raya
COPY --from=builder_core /app/v2raya_core /usr/bin/v2raya_core
COPY --from=ghcr.io/v2fly/v2ray:latest-extra /opt/v2ray/share /usr/share/v2ray/

EXPOSE 2017
VOLUME /etc/v2raya
ENTRYPOINT ["/usr/bin/v2raya"]