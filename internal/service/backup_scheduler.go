package service

import (
	"context"
	"time"
)

// StartAutomaticBackup keeps local JSON installations protected without
// requiring the learner to remember a manual export. PostgreSQL deployments
// are deliberately skipped because every account owns isolated data and a
// process-level scheduler has no authenticated user scope.
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
