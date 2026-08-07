# Docker 部署指南

本部署方案适用于 2 核 2 GiB 的 Ubuntu 22.04 服务器。服务由三个容器组成：

~~~text
Caddy（对外提供 HTTP/HTTPS）
  -> Tracker（Go 后端和已内嵌的 Vue 前端）
  -> PostgreSQL（仅 Docker 内网可访问）
~~~

不要以默认 JSON 模式对公网运行。JSON 模式没有账号隔离；本方案固定使用 PostgreSQL 登录模式。

## 首次部署

1. 在服务器安装 Docker Engine 和 Docker Compose 插件。
2. 将仓库克隆到服务器，例如：

~~~sh
git clone https://github.com/Zilvren/Learning-Assistant.git
cd Learning-Assistant/deploy
~~~

3. 创建生产环境文件：

~~~sh
cp .env.example .env
nano .env
~~~

在 .env 中把 POSTGRES_PASSWORD 和 TRACKER_JWT_SECRET 都改为不同的、至少 32 位的随机字符串。可用下面的命令生成：

~~~sh
openssl rand -hex 32
~~~

首次启动时保留 TRACKER_REGISTRATION_ENABLED=true，先允许创建自己的账户。

4. 启动服务：

~~~sh
docker compose up -d --build
docker compose ps
~~~

浏览器访问 http://服务器公网IP。出现登录页后注册自己的账号。

5. 注册完成后，编辑 .env：

~~~text
TRACKER_REGISTRATION_ENABLED=false
~~~

然后执行：

~~~sh
docker compose up -d app
~~~

这样页面不再显示“注册”，陌生人也无法创建账户。

## 迁移当前电脑上的数据

先在当前本地应用的设置页选择“备份数据”。登录服务器上的账号后，在服务器版的设置页选择“导入备份”，上传刚下载的 zip 文件即可。资料库附件和图片也会一同迁移。

## 域名与 HTTPS

在中国大陆服务器使用域名公开访问前，需要完成 ICP 备案。备案和 DNS 生效后：

1. 将域名的 A 记录指向服务器公网 IP。
2. 把 deploy/Caddyfile 第一行的 :80 改为你的域名，例如 study.example.com。
3. 将 .env 中的 TRACKER_COOKIE_SECURE 改为 true。
4. 重启：

~~~sh
docker compose up -d
~~~

Caddy 会自动申请并续期 HTTPS 证书。

## 日常维护

查看服务状态和日志：

~~~sh
cd ~/Learning-Assistant/deploy
docker compose ps
docker compose logs -f --tail=100
~~~

升级代码：

~~~sh
git pull
docker compose up -d --build
~~~

创建数据库备份：

~~~sh
docker compose exec -T db pg_dump -U study_tracker study_tracker | gzip > study-tracker-$(date +%F).sql.gz
~~~

不要执行 docker compose down -v；其中的 -v 会删除 PostgreSQL 和应用数据卷。
