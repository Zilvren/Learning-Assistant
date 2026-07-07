package service

import (
	"context"
	"strings"

	models "study-tracker-go/internal/model"
)

type TokenInfo struct {
	Token      string `json:"token"`
	Configured bool   `json:"configured"`
	Username   string `json:"username"`
}

func GetTokenInfo(ctx context.Context) (*TokenInfo, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}

	info := &TokenInfo{
		Username:   config.Username,
		Configured: config.MineruToken != "",
	}
	token := strings.TrimSpace(config.MineruToken)
	if token != "" {
		if len(token) > 12 {
			info.Token = token[:8] + "***" + token[len(token)-4:]
		} else {
			info.Token = "***"
		}
	}
	return info, nil
}

func SetToken(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	config, err := loadConfig(ctx)
	if err != nil {
		return err
	}
	config.MineruToken = token
	return saveConfig(ctx, config)
}

func ClearToken(ctx context.Context) error {
	config, err := loadConfig(ctx)
	if err != nil {
		return err
	}
	config.MineruToken = ""
	return saveConfig(ctx, config)
}

func SetUsername(ctx context.Context, name string) error {
	config, err := loadConfig(ctx)
	if err != nil {
		return err
	}
	config.Username = strings.TrimSpace(name)
	return saveConfig(ctx, config)
}

func loadConfig(ctx context.Context) (models.Config, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return models.Config{}, err
	}
	return repos.Settings.Load(ctx)
}

func saveConfig(ctx context.Context, config models.Config) error {
	repos, err := repositories(ctx)
	if err != nil {
		return err
	}
	return repos.Settings.Save(ctx, config)
}
