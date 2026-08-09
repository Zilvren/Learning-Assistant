package service

import (
	"context"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/repository"
	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	"study-tracker-go/pkg/config"
)

type verificationAuthRepository struct {
	user      models.AuthUser
	tokenHash string
}

// CreateUser 在业务层中创建或更新相应状态。
func (r *verificationAuthRepository) CreateUser(_ context.Context, username string, email string, _ string, emailVerified bool) (models.User, error) {
	r.user = models.AuthUser{ID: 42, Username: username, Email: email, EmailVerified: emailVerified, PasswordHash: "hash", Status: "active"}
	return models.User{ID: r.user.ID, Username: username, Email: email, EmailVerified: emailVerified, Status: "active"}, nil
}

// DeleteUnverifiedUser 在业务层中删除、清理或撤销相应状态。
func (r *verificationAuthRepository) DeleteUnverifiedUser(_ context.Context, _ int64) error {
	return nil
}

// FindUserByAccount 在业务层中读取并整理所需数据。
func (r *verificationAuthRepository) FindUserByAccount(_ context.Context, _ string) (models.AuthUser, error) {
	return r.user, nil
}

// FindUserByID 在业务层中读取并整理所需数据。
func (r *verificationAuthRepository) FindUserByID(_ context.Context, _ int64) (models.AuthUser, error) {
	return r.user, nil
}

// CreateEmailVerificationToken 在业务层中创建或更新相应状态。
func (r *verificationAuthRepository) CreateEmailVerificationToken(_ context.Context, _ int64, tokenHash string, _ time.Time) error {
	r.tokenHash = tokenHash
	return nil
}

// ConsumeEmailVerificationToken 在业务层中完成本文件定义的局部处理。
func (r *verificationAuthRepository) ConsumeEmailVerificationToken(_ context.Context, tokenHash string) (models.AuthUser, error) {
	if tokenHash == "" || tokenHash != r.tokenHash {
		return models.AuthUser{}, errors.New("invalid token")
	}
	r.user.EmailVerified = true
	return r.user, nil
}

// TouchLastLogin 在业务层中完成本文件定义的局部处理。
func (r *verificationAuthRepository) TouchLastLogin(_ context.Context, _ int64) error { return nil }

// CreateRefreshToken 在业务层中创建或更新相应状态。
func (r *verificationAuthRepository) CreateRefreshToken(_ context.Context, _ int64, _ string, _ string, _ string, _ time.Time) error {
	return nil
}

// FindRefreshToken 在业务层中读取并整理所需数据。
func (r *verificationAuthRepository) FindRefreshToken(_ context.Context, _ string) (int64, time.Time, bool, error) {
	return 0, time.Time{}, true, errors.New("not implemented")
}

// RevokeRefreshToken 在业务层中删除、清理或撤销相应状态。
func (r *verificationAuthRepository) RevokeRefreshToken(_ context.Context, _ string) error {
	return nil
}

var _ repository.AuthRepository = (*verificationAuthRepository)(nil)

// TestRegistrationRequiresEmailVerificationBeforeIssuingTokens 在业务层中验证对应场景的行为与边界条件。
func TestRegistrationRequiresEmailVerificationBeforeIssuingTokens(t *testing.T) {
	repos := jsonrepo.NewRepositories()
	authRepo := &verificationAuthRepository{}
	repos.Auth = authRepo
	cfg := config.Config{
		StorageDriver:            "postgres",
		AuthEnabled:              true,
		RegistrationEnabled:      true,
		JWTSecret:                "test-secret",
		AccessTokenTTL:           time.Minute,
		RefreshTokenTTL:          time.Hour,
		EmailVerificationEnabled: true,
		PublicURL:                "http://localhost:8000",
		SMTPHost:                 "smtp.example.com",
		SMTPPort:                 465,
		SMTPFrom:                 "mailer@example.com",
		SMTPTLSMode:              "implicit",
		EmailVerificationTTL:     time.Hour,
	}
	if err := InitApp(cfg, repos, nil); err != nil {
		t.Fatal(err)
	}

	previousSMTP := smtpSend
	var message []byte
	smtpSend = func(_ config.Config, _ string, value []byte) error {
		message = value
		return nil
	}
	t.Cleanup(func() { smtpSend = previousSMTP })

	result, err := Register(context.Background(), models.RegisterRequest{
		Username: "verified-user",
		Email:    "USER@example.com",
		Password: "password123",
	}, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.EmailVerificationRequired || result.Email != "user@example.com" || result.TokenPair.AccessToken != "" {
		t.Fatalf("unexpected registration result: %#v", result)
	}
	if authRepo.user.EmailVerified || authRepo.tokenHash == "" {
		t.Fatal("expected an unverified account and a persisted verification token")
	}

	token := verificationTokenFromMessage(t, message)
	pair, err := VerifyEmail(context.Background(), token, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || !authRepo.user.EmailVerified || pair.User.Email != "user@example.com" {
		t.Fatalf("verification did not issue an authenticated session: %#v", pair)
	}
}

// verificationTokenFromMessage 在业务层中完成本文件定义的局部处理。
func verificationTokenFromMessage(t *testing.T, message []byte) string {
	t.Helper()
	parts := strings.Split(string(message), "\r\n\r\n")
	if len(parts) != 2 {
		t.Fatalf("unexpected email message: %q", message)
	}
	body, err := base64.StdEncoding.DecodeString(strings.TrimSpace(parts[1]))
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range strings.Fields(string(body)) {
		if strings.HasPrefix(field, "http") {
			parsed, err := url.Parse(field)
			if err != nil {
				t.Fatal(err)
			}
			return parsed.Query().Get("token")
		}
	}
	t.Fatalf("verification URL missing from email body: %q", body)
	return ""
}
