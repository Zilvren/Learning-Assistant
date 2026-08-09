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
	Enabled                  bool
	RegistrationEnabled      bool
	EmailVerificationEnabled bool
	Secret                   string
	AccessTokenTTL           time.Duration
	RefreshTokenTTL          time.Duration
	CookieSecure             bool
	EmailVerificationTTL     time.Duration
}

type AuthStatus struct {
	Enabled                  bool `json:"enabled"`
	RegistrationEnabled      bool `json:"registration_enabled"`
	EmailVerificationEnabled bool `json:"email_verification_enabled"`
	UpdateEnabled            bool `json:"update_enabled"`
}

type RegistrationResult struct {
	TokenPair                 TokenPair
	EmailVerificationRequired bool
	Email                     string
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

// AuthStatusResponse 在业务层中完成本文件定义的局部处理。
func AuthStatusResponse(ctx context.Context) (AuthStatus, error) {
	cfg, err := currentAuthConfig(ctx)
	if err != nil {
		return AuthStatus{}, err
	}
	return AuthStatus{
		Enabled:                  cfg.Enabled,
		RegistrationEnabled:      cfg.Enabled && cfg.RegistrationEnabled,
		EmailVerificationEnabled: cfg.Enabled && cfg.RegistrationEnabled && cfg.EmailVerificationEnabled,
		UpdateEnabled:            !cfg.Enabled,
	}, nil
}

// UpdateEnabled is limited to the desktop JSON mode. Production releases are
// deployed by the server pipeline instead of downloading a client updater.
// UpdateEnabled 返回当前认证开关是否已启用。
func UpdateEnabled(ctx context.Context) bool {
	app, err := appFor(ctx)
	return err == nil && !app.AuthEnabled()
}

// Register 在业务层中完成本文件定义的局部处理。
func Register(ctx context.Context, req models.RegisterRequest, userAgent string, ipAddress string) (RegistrationResult, error) {
	cfg, err := currentAuthConfig(ctx)
	if err != nil {
		return RegistrationResult{}, err
	}
	if !cfg.Enabled {
		return RegistrationResult{}, fmt.Errorf("当前运行模式未启用登录注册")
	}
	if !cfg.RegistrationEnabled {
		return RegistrationResult{}, fmt.Errorf("当前学习空间暂不开放注册")
	}
	username, email, password, err := validateRegister(req, cfg.EmailVerificationEnabled)
	if err != nil {
		return RegistrationResult{}, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return RegistrationResult{}, err
	}
	repo, err := authRepository(ctx)
	if err != nil {
		return RegistrationResult{}, err
	}
	user, err := repo.CreateUser(ctx, username, email, string(passwordHash), !cfg.EmailVerificationEnabled)
	if err != nil {
		return RegistrationResult{}, friendlyAuthError(err)
	}
	if cfg.EmailVerificationEnabled {
		appConfig, configErr := currentConfig(ctx)
		if configErr != nil {
			return RegistrationResult{}, configErr
		}
		if err := createEmailVerification(ctx, repo, appConfig, user); err != nil {
			_ = repo.DeleteUnverifiedUser(ctx, user.ID)
			return RegistrationResult{}, err
		}
		return RegistrationResult{EmailVerificationRequired: true, Email: user.Email}, nil
	}
	pair, err := issueTokens(ctx, repo, cfg, models.AuthUser{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		EmailVerified: true,
		Status:        user.Status,
	}, userAgent, ipAddress)
	if err != nil {
		return RegistrationResult{}, err
	}
	return RegistrationResult{TokenPair: pair}, nil
}

// VerifyEmail 在业务层中完成本文件定义的局部处理。
func VerifyEmail(ctx context.Context, token string, userAgent string, ipAddress string) (TokenPair, error) {
	cfg, err := currentAuthConfig(ctx)
	if err != nil {
		return TokenPair{}, err
	}
	if !cfg.Enabled || !cfg.EmailVerificationEnabled {
		return TokenPair{}, fmt.Errorf("邮箱验证尚未启用")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return TokenPair{}, fmt.Errorf("验证链接无效")
	}
	repo, err := authRepository(ctx)
	if err != nil {
		return TokenPair{}, err
	}
	user, err := repo.ConsumeEmailVerificationToken(ctx, hashRefreshToken(token))
	if err != nil {
		return TokenPair{}, fmt.Errorf("验证链接无效或已过期")
	}
	return issueTokens(ctx, repo, cfg, user, userAgent, ipAddress)
}

// ResendEmailVerification 在业务层中完成本文件定义的局部处理。
func ResendEmailVerification(ctx context.Context, rawEmail string) error {
	cfg, err := currentAuthConfig(ctx)
	if err != nil {
		return err
	}
	if !cfg.Enabled || !cfg.EmailVerificationEnabled {
		return fmt.Errorf("邮箱验证尚未启用")
	}
	email, err := validateEmailAddress(rawEmail)
	if err != nil {
		return err
	}
	repo, err := authRepository(ctx)
	if err != nil {
		return err
	}
	user, err := repo.FindUserByAccount(ctx, email)
	if err != nil || user.Status != "active" || user.Email != email || user.EmailVerified {
		return nil
	}
	appConfig, err := currentConfig(ctx)
	if err != nil {
		return err
	}
	return createEmailVerification(ctx, repo, appConfig, models.User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Status:   user.Status,
	})
}

// Login 在业务层中完成本文件定义的局部处理。
func Login(ctx context.Context, req models.LoginRequest, userAgent string, ipAddress string) (TokenPair, error) {
	cfg, err := currentAuthConfig(ctx)
	if err != nil {
		return TokenPair{}, err
	}
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
	repo, err := authRepository(ctx)
	if err != nil {
		return TokenPair{}, err
	}
	user, err := repo.FindUserByAccount(ctx, account)
	if err != nil || user.Status != "active" {
		return TokenPair{}, fmt.Errorf("账号或密码错误")
	}
	if cfg.EmailVerificationEnabled && user.Email != "" && !user.EmailVerified {
		return TokenPair{}, fmt.Errorf("邮箱尚未验证，请查收验证邮件")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return TokenPair{}, fmt.Errorf("账号或密码错误")
	}
	return issueTokens(ctx, repo, cfg, user, userAgent, ipAddress)
}

// RefreshLogin 在业务层中完成本文件定义的局部处理。
func RefreshLogin(ctx context.Context, refreshToken string, userAgent string, ipAddress string) (TokenPair, error) {
	cfg, err := currentAuthConfig(ctx)
	if err != nil {
		return TokenPair{}, err
	}
	if !cfg.Enabled {
		return TokenPair{}, fmt.Errorf("当前运行模式未启用登录注册")
	}
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return TokenPair{}, fmt.Errorf("未登录")
	}
	repo, err := authRepository(ctx)
	if err != nil {
		return TokenPair{}, err
	}
	tokenHash := hashRefreshToken(refreshToken)
	userID, expiresAt, revoked, err := repo.FindRefreshToken(ctx, tokenHash)
	if err != nil || revoked || time.Now().After(expiresAt) {
		return TokenPair{}, fmt.Errorf("登录已过期")
	}
	user, err := repo.FindUserByID(ctx, userID)
	if err != nil || user.Status != "active" || (cfg.EmailVerificationEnabled && user.Email != "" && !user.EmailVerified) {
		return TokenPair{}, fmt.Errorf("登录已过期")
	}
	if err := repo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return TokenPair{}, err
	}
	return issueTokens(ctx, repo, cfg, user, userAgent, ipAddress)
}

// Logout 在业务层中完成本文件定义的局部处理。
func Logout(ctx context.Context, refreshToken string) error {
	app, err := appFor(ctx)
	if err != nil || !app.AuthEnabled() || strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	repo, err := authRepository(ctx)
	if err != nil {
		return err
	}
	return repo.RevokeRefreshToken(ctx, hashRefreshToken(refreshToken))
}

// CurrentUser 在业务层中完成本文件定义的局部处理。
func CurrentUser(ctx context.Context) (models.User, error) {
	cfg, err := currentAuthConfig(ctx)
	if err != nil {
		return models.User{}, err
	}
	if !cfg.Enabled {
		return models.User{}, fmt.Errorf("当前运行模式未启用登录注册")
	}
	userID, ok := UserIDFromContext(ctx)
	if !ok || userID <= 0 {
		return models.User{}, fmt.Errorf("未登录")
	}
	repo, err := authRepository(ctx)
	if err != nil {
		return models.User{}, err
	}
	user, err := repo.FindUserByID(ctx, userID)
	if err != nil || user.Status != "active" || (cfg.EmailVerificationEnabled && user.Email != "" && !user.EmailVerified) {
		return models.User{}, fmt.Errorf("未登录")
	}
	return publicUser(user), nil
}

// ValidateAccessToken 在业务层中校验输入或判断当前条件。
func ValidateAccessToken(ctx context.Context, token string) (models.AuthUser, error) {
	cfg, err := currentAuthConfig(ctx)
	if err != nil {
		return models.AuthUser{}, err
	}
	if !cfg.Enabled {
		return models.AuthUser{}, fmt.Errorf("当前运行模式未启用登录注册")
	}
	claims, err := parseAccessToken(token, cfg.Secret)
	if err != nil {
		return models.AuthUser{}, err
	}
	repo, err := authRepository(ctx)
	if err != nil {
		return models.AuthUser{}, err
	}
	user, err := repo.FindUserByID(ctx, claims.Sub)
	if err != nil || user.Status != "active" || (cfg.EmailVerificationEnabled && user.Email != "" && !user.EmailVerified) {
		return models.AuthUser{}, fmt.Errorf("未登录")
	}
	return user, nil
}

// AccessTokenTTL 在业务层中完成本文件定义的局部处理。
func AccessTokenTTL(ctx context.Context) time.Duration {
	cfg, _ := currentAuthConfig(ctx)
	return cfg.AccessTokenTTL
}

// RefreshTokenTTL 在业务层中完成本文件定义的局部处理。
func RefreshTokenTTL(ctx context.Context) time.Duration {
	cfg, _ := currentAuthConfig(ctx)
	return cfg.RefreshTokenTTL
}

// CookieSecure 在业务层中完成本文件定义的局部处理。
func CookieSecure(ctx context.Context) bool {
	cfg, _ := currentAuthConfig(ctx)
	return cfg.CookieSecure
}

// currentAuthConfig 在业务层中完成本文件定义的局部处理。
func currentAuthConfig(ctx context.Context) (AuthConfig, error) {
	appConfig, err := currentConfig(ctx)
	if err != nil {
		return AuthConfig{}, err
	}
	return AuthConfig{
		Enabled:                  appConfig.AuthEnabled,
		RegistrationEnabled:      appConfig.RegistrationEnabled,
		EmailVerificationEnabled: appConfig.EmailVerificationEnabled,
		Secret:                   appConfig.JWTSecret,
		AccessTokenTTL:           appConfig.AccessTokenTTL,
		RefreshTokenTTL:          appConfig.RefreshTokenTTL,
		CookieSecure:             appConfig.CookieSecure,
		EmailVerificationTTL:     appConfig.EmailVerificationTTL,
	}, nil
}

// validateRegister 在业务层中校验输入或判断当前条件。
func validateRegister(req models.RegisterRequest, requireEmail bool) (string, string, string, error) {
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
	if requireEmail && email == "" {
		return "", "", "", fmt.Errorf("请填写用于验证账号的邮箱")
	}
	if email != "" {
		normalizedEmail, validationErr := validateEmailAddress(email)
		if validationErr != nil {
			return "", "", "", validationErr
		}
		email = normalizedEmail
	}
	if len(password) < 8 || len(password) > 128 {
		return "", "", "", fmt.Errorf("密码长度需要在 8 到 128 位之间")
	}
	return username, email, password, nil
}

// issueTokens 在业务层中校验输入或判断当前条件。
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

// publicUser 在业务层中完成本文件定义的局部处理。
func publicUser(user models.AuthUser) models.User {
	return models.User{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Status:        user.Status,
	}
}

// buildAccessToken 在业务层中构造、编码或标准化数据。
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

// parseAccessToken 在业务层中解析外部输入为内部数据。
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

// signJWT 在业务层中构造、编码或标准化数据。
func signJWT(unsigned string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// randomToken 在业务层中完成本文件定义的局部处理。
func randomToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

// hashRefreshToken 在业务层中校验输入或判断当前条件。
func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// friendlyAuthError 在业务层中完成本文件定义的局部处理。
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
