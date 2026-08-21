# DaoCloud 代理让中国大陆部署不依赖 Docker Hub 访问。
FROM docker.m.daocloud.io/library/node:22-bookworm-slim AS frontend-builder
WORKDIR /src/frontend

COPY frontend/package.json frontend/package-lock.json ./
# 镜像在 GitHub 托管的运行器上构建，官方 npm 仓库比大陆镜像更可靠；npm 仍会校验锁定文件的完整性哈希。
RUN npm ci --no-audit --no-fund --registry=https://registry.npmjs.org

COPY frontend/ ./
ENV NODE_OPTIONS=--max-old-space-size=512
RUN npm run build

# 生产应用可选地将 DeepSeek Harness 作为子进程运行。单独安装其固定版本运行时，确保最终镜像只包含运行时依赖而不包含本仓库的开发工具。
FROM docker.m.daocloud.io/library/node:22-bookworm-slim AS harness-builder
WORKDIR /app/harness

COPY harness/package.json harness/package-lock.json ./
RUN npm ci --omit=dev --no-audit --no-fund --registry=https://registry.npmjs.org
COPY harness/ ./

FROM docker.m.daocloud.io/library/golang:1.26-bookworm AS app-builder
WORKDIR /src

COPY go.mod go.sum ./
# 同时使用大陆校验和服务，并保留 Go 的模块完整性校验而不将其关闭。
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn
RUN go mod download

COPY . ./
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

# 限制编译器并发度，使一次性构建可安全运行在 2 GiB 内存的服务器上。
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOMAXPROCS=1 \
    go build -trimpath -ldflags="-s -w" -o /out/tracker .

# 最终镜像保留 Node.js 22：Go 服务始终通过受限的 Harness JSON-RPC Agent 提供 AI 对话。
FROM docker.m.daocloud.io/library/node:22-bookworm-slim
WORKDIR /app
# Go 构建镜像已包含证书包；直接复制它，避免在运行时镜像构建期间访问 Debian 软件包镜像。
COPY --from=app-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=app-builder /out/tracker /app/tracker
COPY --from=harness-builder --chown=10001:10001 /app/harness /app/harness
COPY --from=app-builder --chown=10001:10001 /src/packages/dsh-learning-library /app/packages/dsh-learning-library
RUN mkdir /app/data && chown -R 10001:10001 /app

USER 10001:10001
EXPOSE 8000

ENTRYPOINT ["/app/tracker", "--no-browser"]
