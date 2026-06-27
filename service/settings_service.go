package service

import (
	"strings"

	"study-tracker-go/models"
	"study-tracker-go/store"
)

// GetTokenInfo 获取 Token 信息（已脱敏）
type TokenInfo struct {
	Token      string `json:"token"`
	Configured bool   `json:"configured"`
	Username   string `json:"username"`
}

func GetTokenInfo() (*TokenInfo, error) {
	config, err := loadConfig()
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

// SetToken 设置 Token
func SetToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	config, err := loadConfig()
	if err != nil {
		return err
	}
	config.MineruToken = token
	return store.SaveJSON("config.json", config)
}

// ClearToken 清除 Token
func ClearToken() error {
	config, err := loadConfig()
	if err != nil {
		return err
	}
	config.MineruToken = ""
	return store.SaveJSON("config.json", config)
}

// SetUsername 设置用户名
func SetUsername(name string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}
	config.Username = strings.TrimSpace(name)
	return store.SaveJSON("config.json", config)
}

func loadConfig() (models.Config, error) {
	var config models.Config
	if err := store.LoadJSON("config.json", &config); err != nil {
		return models.Config{}, err
	}
	return config, nil
}
