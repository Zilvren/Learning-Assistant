# DaoCloud proxy keeps China-based deployments independent of Docker Hub access.
FROM docker.m.daocloud.io/library/node:22-bookworm-slim AS frontend-builder
WORKDIR /src/frontend

COPY frontend/package.json frontend/package-lock.json ./
# The image is built on GitHub-hosted runners, where the official npm registry
# is more reliable than a mainland mirror. npm still verifies the lockfile's
# integrity hashes.
RUN npm ci --no-audit --no-fund --registry=https://registry.npmjs.org

COPY frontend/ ./
ENV NODE_OPTIONS=--max-old-space-size=512
RUN npm run build

# The production app can optionally run DeepSeek Harness as a child process.
# Install its pinned runtime separately so the final image contains only the
# dependencies needed at runtime, not this repository's development tooling.
FROM docker.m.daocloud.io/library/node:22-bookworm-slim AS harness-builder
WORKDIR /app/harness

COPY harness/package.json harness/package-lock.json ./
RUN npm ci --omit=dev --no-audit --no-fund --registry=https://registry.npmjs.org
COPY harness/ ./

FROM docker.m.daocloud.io/library/golang:1.26-bookworm AS app-builder
WORKDIR /src

COPY go.mod go.sum ./
# Use the mainland checksum endpoint as well, while retaining Go's module
# integrity verification rather than disabling it.
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn
RUN go mod download

COPY . ./
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

# Limit compiler concurrency so a one-time build is safe on a 2 GiB server.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOMAXPROCS=1 \
    go build -trimpath -ldflags="-s -w" -o /out/tracker .

# Keep Node.js 22 in the final image: the Go service starts the restricted
# Harness JSON-RPC agent only when STUDY_HARNESS_ENABLED=true.
FROM docker.m.daocloud.io/library/node:22-bookworm-slim
WORKDIR /app
# The Go builder already contains the certificate bundle.  Copy it instead of
# reaching Debian package mirrors during the runtime-image build.
COPY --from=app-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=app-builder /out/tracker /app/tracker
COPY --from=harness-builder --chown=10001:10001 /app/harness /app/harness
COPY --from=app-builder --chown=10001:10001 /src/packages/dsh-learning-library /app/packages/dsh-learning-library
RUN mkdir /app/data && chown -R 10001:10001 /app

USER 10001:10001
EXPOSE 8000

ENTRYPOINT ["/app/tracker", "--no-browser"]
