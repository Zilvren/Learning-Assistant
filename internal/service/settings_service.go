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
	Model      string `json:"model,omitempty"`
}

// GetTokenInfo 在业务层中读取并整理所需数据。
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

// SetToken 在业务层中完成本文件定义的局部处理。
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

// ClearToken 在业务层中删除、清理或撤销相应状态。
func ClearToken(ctx context.Context) error {
	config, err := loadConfig(ctx)
	if err != nil {
		return err
	}
	config.MineruToken = ""
	return saveConfig(ctx, config)
}

// GetDeepSeekTokenInfo 仅返回脱敏后的状态，绝不会将完整 API 密钥返回给浏览器。
func GetDeepSeekTokenInfo(ctx context.Context) (*TokenInfo, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	modelName, err := deepSeekModel(ctx)
	if err != nil {
		return nil, err
	}
	return &TokenInfo{Token: maskToken(config.DeepSeekToken), Configured: strings.TrimSpace(config.DeepSeekToken) != "", Model: modelName}, nil
}

// SetDeepSeekToken 在业务层中执行当前流程或局部处理。
func SetDeepSeekToken(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	config, err := loadConfig(ctx)
	if err != nil {
		return err
	}
	config.DeepSeekToken = token
	return saveConfig(ctx, config)
}

// ClearDeepSeekToken 在业务层中执行当前流程或局部处理。
func ClearDeepSeekToken(ctx context.Context) error {
	config, err := loadConfig(ctx)
	if err != nil {
		return err
	}
	config.DeepSeekToken = ""
	return saveConfig(ctx, config)
}

// SetDeepSeekModel 在业务层中执行当前流程或局部处理。
func SetDeepSeekModel(ctx context.Context, modelName string) (string, error) {
	modelName, err := normalizeDeepSeekModel(modelName)
	if err != nil {
		return "", err
	}
	config, err := loadConfig(ctx)
	if err != nil {
		return "", err
	}
	config.DeepSeekModel = modelName
	if err := saveConfig(ctx, config); err != nil {
		return "", err
	}
	return modelName, nil
}

// maskToken 在业务层中执行当前流程或局部处理。
func maskToken(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 12 {
		return value[:8] + "***" + value[len(value)-4:]
	}
	if value != "" {
		return "***"
	}
	return ""
}

// SetUsername 在业务层中完成本文件定义的局部处理。
func SetUsername(ctx context.Context, name string) error {
	config, err := loadConfig(ctx)
	if err != nil {
		return err
	}
	config.Username = strings.TrimSpace(name)
	return saveConfig(ctx, config)
}

// loadConfig 在业务层中读取并整理所需数据。
func loadConfig(ctx context.Context) (models.Config, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return models.Config{}, err
	}
	return repos.Settings.Load(ctx)
}

// saveConfig 在业务层中创建或更新相应状态。
func saveConfig(ctx context.Context, config models.Config) error {
	repos, err := repositories(ctx)
	if err != nil {
		return err
	}
	return repos.Settings.Save(ctx, config)
}
