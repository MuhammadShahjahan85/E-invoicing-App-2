# Veridian E-invoicing Pakistan — container image.
#
#   docker build --build-arg VERSION=1.0.0 --build-arg LICENSE_PUBKEY=<base64> -t veridian-einvoicing .
#   docker run -d --name einvoice --restart unless-stopped -p 8443:8443 -v einvoice-data:/data veridian-einvoicing
#
# Back up the /data volume (database, master.key, certificates, backups).

FROM node:20-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
# vite.config.ts writes the bundle to ../internal/webui/dist
RUN mkdir -p /src/internal/webui && npm run build

FROM golang:1.24-alpine AS build
ARG VERSION=1.0.0
ARG BUILD_DATE=unknown
ARG LICENSE_PUBKEY=
WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/webui/dist ./internal/webui/dist
RUN go build -trimpath \
    -ldflags "-s -w -X einvoicing/internal/brand.Version=${VERSION} -X einvoicing/internal/brand.BuildDate=${BUILD_DATE} -X einvoicing/internal/license.PublicKeyB64=${LICENSE_PUBKEY}" \
    -o /out/einvoice ./cmd/einvoice

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 einvoice \
    && mkdir -p /data && chown einvoice:einvoice /data
COPY --from=build /out/einvoice /app/einvoice
COPY LICENSE THIRD-PARTY-NOTICES.txt /app/
USER einvoice
ENV TZ=Asia/Karachi
VOLUME /data
EXPOSE 8443
ENTRYPOINT ["/app/einvoice", "serve", "--data", "/data"]
