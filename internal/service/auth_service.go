package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	models "study-tracker-go/internal/model"

	"golang.org/x/crypto/bcrypt"
)

const (
	AccessCookieName  = "tracker_access"
	RefreshCookieName = "tracker_refresh"
)

type AuthConfig struct {
	Enabled         bool
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CookieSecure    bool
}

type AuthStatus struct {
	Enabled bool `json:"enabled"`
}

type TokenPair struct {
	AccessToken          string
	RefreshToken         string
	AccessTokenExpiresAt time.Time
	RefreshExpiresAt     time.Time
	User                 models.User
}

type jwtClaims struct {
	Sub      int64  `json:"sub"`
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}

func AuthStatusResponse() AuthStatus {
	return AuthStatus{Enabled: AuthEnabled()}
}

func Register(ctx context.Context, req models.RegisterRequest, userAgent string, ipAddress string) (TokenPair, error) {
	cfg := currentAuthConfig()
	if !cfg.Enabled {
		return TokenPair{}, fmt.Errorf("当前运行模式未启用登录注册")
	}
	username, email, password, err := validateRegister(req)
	if err != nil {
		return TokenPair{}, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return TokenPair{}, err
	}
	repo, err := authRepository()
	if err != nil {
		return TokenPair{}, err
	}
	user, err := repo.CreateUser(ctx, username, email, string(passwordHash))
	if err != nil {
		return TokenPair{}, friendlyAuthError(err)
	}
	return issueTokens(ctx, repo, cfg, models.AuthUser{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Status:   user.Status,
	}, userAgent, ipAddress)
}

func Login(ctx context.Context, req models.LoginRequest, userAgent string, ipAddress string) (TokenPair, error) {
	cfg := currentAuthConfig()
	if !cfg.Enabled {
		return TokenPair{}, fmt.Errorf("当前运行模式未启用登录注册")
	}
	account := strings.TrimSpace(req.Account)
	if account == "" {
		return TokenPair{}, fmt.Errorf("请输入用户名或邮箱")
	}
	if strings.TrimSpace(req.Password) == "" {
		return TokenPair{}, fmt.Errorf("请输入密码")
	}
	repo, err := authRepository()
	if err != nil {
		return TokenPair{}, err
	}
	user, err := repo.FindUserByAccount(ctx, account)
	if err != nil || user.Status != "active" {
		return TokenPair{}, fmt.Errorf("账号或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return TokenPair{}, fmt.Errorf("账号或密码错误")
	}
	return issueTokens(ctx, repo, cfg, user, userAgent, ipAddress)
}

func RefreshLogin(ctx context.Context, refreshToken string, userAgent string, ipAddress string) (TokenPair, error) {
	cfg := currentAuthConfig()
	if !cfg.Enabled {
		return TokenPair{}, fmt.Errorf("当前运行模式未启用登录注册")
	}
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return TokenPair{}, fmt.Errorf("未登录")
	}
	repo, err := authRepository()
	if err != nil {
		return TokenPair{}, err
	}
	tokenHash := hashRefreshToken(refreshToken)
	userID, expiresAt, revoked, err := repo.FindRefreshToken(ctx, tokenHash)
	if err != nil || revoked || time.Now().After(expiresAt) {
		return TokenPair{}, fmt.Errorf("登录已过期")
	}
	user, err := repo.FindUserByID(ctx, userID)
	if err != nil || user.Status != "active" {
		return TokenPair{}, fmt.Errorf("登录已过期")
	}
	if err := repo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return TokenPair{}, err
	}
	return issueTokens(ctx, repo, cfg, user, userAgent, ipAddress)
}

func Logout(ctx context.Context, refreshToken string) error {
	if !AuthEnabled() || strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	repo, err := authRepository()
	if err != nil {
		return err
	}
	return repo.RevokeRefreshToken(ctx, hashRefreshToken(refreshToken))
}

func CurrentUser(ctx context.Context) (models.User, error) {
	cfg := currentAuthConfig()
	if !cfg.Enabled {
		return models.User{}, fmt.Errorf("当前运行模式未启用登录注册")
	}
	userID, ok := UserIDFromContext(ctx)
	if !ok || userID <= 0 {
		return models.User{}, fmt.Errorf("未登录")
	}
	repo, err := authRepository()
	if err != nil {
		return models.User{}, err
	}
	user, err := repo.FindUserByID(ctx, userID)
	if err != nil || user.Status != "active" {
		return models.User{}, fmt.Errorf("未登录")
	}
	return publicUser(user), nil
}

func ValidateAccessToken(token string) (models.AuthUser, error) {
	cfg := currentAuthConfig()
	if !cfg.Enabled {
		return models.AuthUser{}, fmt.Errorf("当前运行模式未启用登录注册")
	}
	claims, err := parseAccessToken(token, cfg.Secret)
	if err != nil {
		return models.AuthUser{}, err
	}
	repo, err := authRepository()
	if err != nil {
		return models.AuthUser{}, err
	}
	user, err := repo.FindUserByID(context.Background(), claims.Sub)
	if err != nil || user.Status != "active" {
		return models.AuthUser{}, fmt.Errorf("未登录")
	}
	return user, nil
}

func AccessTokenTTL() time.Duration {
	return currentAuthConfig().AccessTokenTTL
}

func RefreshTokenTTL() time.Duration {
	return currentAuthConfig().RefreshTokenTTL
}

func CookieSecure() bool {
	return currentAuthConfig().CookieSecure
}

func currentAuthConfig() AuthConfig {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return AuthConfig{
		Enabled:         appConfig.AuthEnabled,
		Secret:          appConfig.JWTSecret,
		AccessTokenTTL:  appConfig.AccessTokenTTL,
		RefreshTokenTTL: appConfig.RefreshTokenTTL,
		CookieSecure:    appConfig.CookieSecure,
	}
}

func validateRegister(req models.RegisterRequest) (string, string, string, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password
	if len([]rune(username)) < 3 || len([]rune(username)) > 32 {
		return "", "", "", fmt.Errorf("用户名长度需要在 3 到 32 个字符之间")
	}
	for _, r := range username {
		if !(r == '_' || r == '-' || r == '.' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= 0x4e00 && r <= 0x9fff) {
			return "", "", "", fmt.Errorf("用户名只能包含中文、字母、数字、下划线、短横线和点")
		}
	}
	if email != "" && !strings.Contains(email, "@") {
		return "", "", "", fmt.Errorf("邮箱格式不正确")
	}
	if len(password) < 8 || len(password) > 128 {
		return "", "", "", fmt.Errorf("密码长度需要在 8 到 128 位之间")
	}
	return username, email, password, nil
}

func issueTokens(ctx context.Context, repo interface {
	CreateRefreshToken(context.Context, int64, string, string, string, time.Time) error
	TouchLastLogin(context.Context, int64) error
}, cfg AuthConfig, user models.AuthUser, userAgent string, ipAddress string) (TokenPair, error) {
	if user.Status != "active" {
		return TokenPair{}, fmt.Errorf("账号不可用")
	}
	now := time.Now()
	accessExpiresAt := now.Add(cfg.AccessTokenTTL)
	refreshExpiresAt := now.Add(cfg.RefreshTokenTTL)
	accessToken, err := buildAccessToken(jwtClaims{
		Sub:      user.ID,
		Username: user.Username,
		Iat:      now.Unix(),
		Exp:      accessExpiresAt.Unix(),
	}, cfg.Secret)
	if err != nil {
		return TokenPair{}, err
	}
	refreshToken, err := randomToken(32)
	if err != nil {
		return TokenPair{}, err
	}
	if err := repo.CreateRefreshToken(ctx, user.ID, hashRefreshToken(refreshToken), userAgent, ipAddress, refreshExpiresAt); err != nil {
		return TokenPair{}, err
	}
	_ = repo.TouchLastLogin(ctx, user.ID)
	return TokenPair{
		AccessToken:          accessToken,
		RefreshToken:         refreshToken,
		AccessTokenExpiresAt: accessExpiresAt,
		RefreshExpiresAt:     refreshExpiresAt,
		User:                 publicUser(user),
	}, nil
}

func publicUser(user models.AuthUser) models.User {
	return models.User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Status:   user.Status,
	}
}

func buildAccessToken(claims jwtClaims, secret string) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	signature := signJWT(unsigned, secret)
	return unsigned + "." + signature, nil
}

func parseAccessToken(token string, secret string) (jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}, fmt.Errorf("未登录")
	}
	unsigned := parts[0] + "." + parts[1]
	expected := signJWT(unsigned, secret)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return jwtClaims{}, fmt.Errorf("未登录")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtClaims{}, fmt.Errorf("未登录")
	}
	var claims jwtClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return jwtClaims{}, fmt.Errorf("未登录")
	}
	if claims.Sub <= 0 || time.Now().Unix() >= claims.Exp {
		return jwtClaims{}, fmt.Errorf("未登录")
	}
	return claims, nil
}

func signJWT(unsigned string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func friendlyAuthError(err error) error {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "uq_users_username_active") {
		return fmt.Errorf("用户名已存在")
	}
	if strings.Contains(message, "uq_users_email_active") {
		return fmt.Errorf("邮箱已存在")
	}
	return err
}
