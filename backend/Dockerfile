FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata ffmpeg python3 py3-pip nodejs && \
    pip3 install --no-cache-dir --break-system-packages "yt-dlp[default]" && \
    adduser -D -u 10001 api && \
    mkdir -p /data && chown api /data

WORKDIR /srv
COPY --from=build /out/api /srv/api

USER api
EXPOSE 8000

ENTRYPOINT ["/srv/api"]