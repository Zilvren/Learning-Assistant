# 服务器部署与日常维护指南

这份文档适用于本项目部署在 Ubuntu 22.04 的小规格服务器（建议至少 2 核 2 GiB，并配置 2 GiB Swap）。当前方案使用 Docker Compose，运行三个容器：

```text
互联网
  -> Caddy（HTTP/HTTPS 入口）
  -> Tracker（Go 后端与内嵌的 Vue 前端）
  -> PostgreSQL（仅 Docker 内网可访问）
```

不要把默认 JSON 存储模式直接暴露到公网：它没有账号隔离。本部署固定使用 PostgreSQL 登录模式。

## 服务器信息与安全组

安全组只需要如下入方向规则：

| 用途 | 协议 / 端口 | 来源 |
| --- | --- | --- |
| 网站 HTTP | TCP 80 | `0.0.0.0/0` |
| 网站 HTTPS | TCP 443 | `0.0.0.0/0` |
| 管理 SSH | TCP 22 | 管理者当前公网 IP，例如 `203.0.113.8/32` |

不要保留 `SSH(22) -> 0.0.0.0/0`、RDP(3389) 或“全部 ICMP”这类不需要的公网规则。服务器上不需要对公网开放 PostgreSQL 的 5432 端口。

## 首次部署

### 1. 安装 Docker

在服务器上执行：

```bash
apt update
apt install -y docker.io docker-compose-v2
systemctl enable --now docker
docker --version
docker compose version
```

若 Docker Hub 在服务器上无法访问，项目的 `Dockerfile` 和 Compose 文件已明确使用 DaoCloud 镜像代理，无需购买 ACR。不要重复安装控制台中的“Docker 社区版”扩展。

### 2. 上传项目

若服务器无法直接克隆 GitHub 仓库，可从本地电脑打包并上传。以下命令在本地项目根目录执行：

```powershell
git archive --format=tar.gz -o Learning-Assistant.tar.gz main
scp .\Learning-Assistant.tar.gz root@<服务器公网IP>:/opt/
```

在服务器执行：

```bash
mkdir -p /opt/Learning-Assistant
tar -xzf /opt/Learning-Assistant.tar.gz -C /opt/Learning-Assistant
rm /opt/Learning-Assistant.tar.gz
```

### 3. 创建生产配置

```bash
cd /opt/Learning-Assistant/deploy
cp .env.example .env
chmod 600 .env
```

编辑 `.env`，将 `POSTGRES_PASSWORD` 和 `TRACKER_JWT_SECRET` 替换为两个不同的强随机值。可用：

```bash
openssl rand -hex 32
```

首次启动时保持：

```text
TRACKER_REGISTRATION_ENABLED=true
```

### 4. 启动与验证

```bash
docker compose up -d --build
docker compose ps
```

三个服务应处于以下状态：

- `db`：`healthy`
- `app`：`Up`
- `caddy`：`Up`，并显示 `0.0.0.0:80->80/tcp`

浏览器访问 `http://<服务器公网IP>`。第一次只用作连通性测试，不要使用复用的重要密码。

注册自己的账号后，为避免陌生人继续注册，编辑 `.env`：

```text
TRACKER_REGISTRATION_ENABLED=false
```

然后重启应用容器：

```bash
docker compose up -d app
```

## 以后修改 Bug 或发布新版本

**不要直接在容器里改文件。** 容器重建后，所有容器内的手工修改都会消失；本地项目和 GitHub 才是代码的唯一来源。

日常流程如下：

1. 在本地修改并测试代码，提交并推送 GitHub。
2. 将本次变动的文件上传到服务器相同的目录。
3. 只重建应用容器。

例如修改了 `api/handlers/example.go`，在本地 PowerShell 中执行：

```powershell
scp .\api\handlers\example.go root@<服务器公网IP>:/opt/Learning-Assistant/api/handlers/example.go
```

然后在服务器执行：

```bash
cd /opt/Learning-Assistant/deploy
docker compose up -d --build app
docker compose ps
```

这不会删除 PostgreSQL 数据或学习资料。若一次修改涉及许多文件，使用“上传项目”中的 `git archive` 方式重新上传源码；`.env` 不在 Git 中，不会被该压缩包覆盖。

## 数据库迁移与数据安全

PostgreSQL 数据存储在 Docker 卷 `deploy_postgres_data` 中。

```bash
# 备份数据库（在 deploy 目录执行）
docker compose exec -T db pg_dump -U study_tracker study_tracker | gzip > study-tracker-$(date +%F).sql.gz
```

以下规则很重要：

- 正式使用后，**不要执行** `docker compose down -v`；`-v` 会删除数据库和资料数据卷。
- 新增数据库表、字段或索引时，请先备份，再新增有序、可重复执行的 `migrations/*.sql` 文件。应用会在启动时依据 `schema_migrations` 自动执行未应用的迁移；不要手工在容器初始化目录重复挂载同一套 SQL。
- 仅当“首次部署、从未注册用户、数据库初始化已明确失败”时，才可以重置数据库卷：

  ```bash
  docker compose down
  docker volume rm deploy_postgres_data
  docker compose up -d
  ```

## 常用排查命令

```bash
cd /opt/Learning-Assistant/deploy

# 服务状态
docker compose ps

# 查看全部日志
docker compose logs --tail=100

# 查看某个服务的日志
docker compose logs --tail=100 app
docker compose logs --tail=100 db
docker compose logs --tail=100 caddy

# 重新构建应用（不删除数据库）
docker compose up -d --build app
```

## 域名与 HTTPS

当前 `Caddyfile` 只提供 HTTP。使用域名正式开放前，先完成适用的备案与 DNS 配置：

1. 将域名的 A 记录指向服务器公网 IP。
2. 将 `deploy/Caddyfile` 第一行的 `:80` 改为你的域名，例如 `study.example.com`。
3. 在 `.env` 中设置 `TRACKER_COOKIE_SECURE=true`。
4. 执行 `docker compose up -d`。

Caddy 会自动申请和续期 HTTPS 证书。HTTPS 开启后再使用正式登录密码。

## 注册邮箱验证

邮箱验证使用验证链接激活新账号。为避免验证令牌在公网 HTTP 链路中泄露，必须先完成上面的域名与 HTTPS 配置，再在 `deploy/.env` 中填写 SMTP 信息并启用：

```text
TRACKER_EMAIL_VERIFICATION_ENABLED=true
TRACKER_PUBLIC_URL=https://你的域名
TRACKER_SMTP_HOST=smtp.example.com
TRACKER_SMTP_PORT=465
TRACKER_SMTP_USERNAME=你的邮箱
TRACKER_SMTP_PASSWORD=邮箱 SMTP 授权码
TRACKER_SMTP_FROM=你的邮箱
TRACKER_SMTP_TLS_MODE=implicit
```

常见的端口组合是 `465 + implicit` 或 `587 + starttls`。保存后执行 `docker compose up -d app`。新注册用户将收到 24 小时有效的验证链接；已有账号会保留现有访问权限。

## 常见问题

### Docker 镜像或依赖下载超时

项目已将容器镜像切换到 DaoCloud，并在 Docker 构建中使用国内 Go 与 npm 依赖代理。若仍失败，先复制完整错误日志，不要反复重装 Docker。

### 数据库显示 unhealthy

先执行：

```bash
docker compose logs --tail=100 db
```

首次初始化失败时，不要让应用带着不完整的表结构继续运行；按上面的“数据库迁移与数据安全”规则处理。

### SSH 上传没有出现密码提示

先确认端口连通：

```powershell
Test-NetConnection <服务器公网IP> -Port 22
```

若显示 `TcpTestSucceeded : True`，再使用：

```powershell
scp -o ConnectTimeout=10 .\文件路径 root@<服务器公网IP>:/目标路径
```
