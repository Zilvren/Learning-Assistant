# syntax=docker/dockerfile:1

FROM node:22-bookworm-slim AS frontend-builder
WORKDIR /src/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY frontend/ ./
ENV NODE_OPTIONS=--max-old-space-size=512
RUN npm run build

FROM golang:1.26-bookworm AS app-builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

# Limit compiler concurrency so a one-time build is safe on a 2 GiB server.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOMAXPROCS=1 \
    go build -trimpath -ldflags="-s -w" -o /out/tracker .

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 tracker \
    && useradd --system --uid 10001 --gid tracker --create-home tracker

WORKDIR /app
COPY --from=app-builder /out/tracker /app/tracker
RUN mkdir /app/data && chown -R tracker:tracker /app

USER tracker
EXPOSE 8000

ENTRYPOINT ["/app/tracker", "--no-browser"]
