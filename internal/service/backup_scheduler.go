package service

import (
	"context"
	"time"
)

// StartAutomaticBackup 为本地 JSON 安装提供自动保护，免去学习者手动导出的负担。PostgreSQL 部署会被跳过，因为每个账户拥有隔离数据，而进程级调度器没有已认证的用户范围。
func StartAutomaticBackup(app *App) func() {
	if app == nil || app.AuthEnabled() {
		return func() {}
	}
	cfg := app.Config()
	if !cfg.AutoBackup || cfg.AutoBackupInterval <= 0 {
		return func() {}
	}
	ctx, cancel := context.WithCancel(ContextWithApp(context.Background(), app))
	go func() {
		_, _ = SaveAutomaticBackupSnapshot(ctx, cfg.AutoBackupKeep)
		ticker := time.NewTicker(cfg.AutoBackupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = SaveAutomaticBackupSnapshot(ctx, cfg.AutoBackupKeep)
			}
		}
	}()
	return cancel
}
