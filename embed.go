package main

import "embed"

// 必须使用 all: 前缀，因为 Vite 会生成以下划线开头的辅助分块，而目录模式嵌入默认会跳过它们。
//
//go:embed all:frontend/dist
var frontendFS embed.FS
