package service

import (
	"context"
	"testing"

	base "study-tracker-go/internal/repository"
	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	"study-tracker-go/pkg/config"
)

// TestRequestAppTakesPriorityOverLegacyApp 在业务层中验证对应场景的行为与边界条件。
func TestRequestAppTakesPriorityOverLegacyApp(t *testing.T) {
	previousDir := base.DataDir()
	base.SetDataDir(t.TempDir())
	t.Cleanup(func() { base.SetDataDir(previousDir) })
	legacy, err := NewApp(config.Config{StorageDriver: "json"}, jsonrepo.NewRepositories(), nil)
	if err != nil {
		t.Fatal(err)
	}
	requestApp, err := NewApp(config.Config{StorageDriver: "json"}, jsonrepo.NewRepositories(), nil)
	if err != nil {
		t.Fatal(err)
	}
	previous := DefaultApp()
	legacyApp.Store(legacy)
	t.Cleanup(func() { legacyApp.Store(previous) })

	resolved, err := appFor(ContextWithApp(context.Background(), requestApp))
	if err != nil {
		t.Fatal(err)
	}
	if resolved != requestApp {
		t.Fatal("request-scoped app must take priority over the legacy fallback")
	}
}

// TestNewAppRejectsIncompleteDependencies 在业务层中验证对应场景的行为与边界条件。
func TestNewAppRejectsIncompleteDependencies(t *testing.T) {
	if _, err := NewApp(config.Config{StorageDriver: "json"}, base.Repositories{}, nil); err == nil {
		t.Fatal("expected incomplete dependencies to be rejected")
	}
}
