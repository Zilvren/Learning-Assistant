package migrations

import "embed"

// FS 将部署迁移同时保存在编译后的应用和 Docker 初始化目录中，使 PostgreSQL 升级不再依赖全新的数据库卷。
//
//go:embed *.sql
var FS embed.FS
