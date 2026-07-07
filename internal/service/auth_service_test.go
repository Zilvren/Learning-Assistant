package service

import (
	"testing"
	"time"

	models "study-tracker-go/internal/model"
	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	"study-tracker-go/pkg/config"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	claims := jwtClaims{
		Sub:      42,
		Username: "tester",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Minute).Unix(),
	}
	token, err := buildAccessToken(claims, "secret")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseAccessToken(token, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Sub != claims.Sub || parsed.Username != claims.Username {
		t.Fatalf("unexpected claims: %#v", parsed)
	}
	if _, err := parseAccessToken(token, "wrong-secret"); err == nil {
		t.Fatal("expected wrong secret to fail")
	}
}

func TestValidateRegister(t *testing.T) {
	username, email, password, err := validateRegister(models.RegisterRequest{
		Username: "test_user",
		Email:    "USER@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if username != "test_user" || email != "user@example.com" || password != "password123" {
		t.Fatalf("unexpected normalized values: %q %q %q", username, email, password)
	}
	if _, _, _, err := validateRegister(models.RegisterRequest{Username: "ab", Password: "password123"}); err == nil {
		t.Fatal("expected short username to fail")
	}
	if _, _, _, err := validateRegister(models.RegisterRequest{Username: "valid", Password: "short"}); err == nil {
		t.Fatal("expected short password to fail")
	}
}

func TestJSONModeAuthDisabled(t *testing.T) {
	if err := InitApp(config.Config{StorageDriver: "json", AuthEnabled: false}, jsonrepo.NewRepositories(), nil); err != nil {
		t.Fatal(err)
	}
	if AuthEnabled() {
		t.Fatal("expected json auth disabled")
	}
	repos, err := repositories(background())
	if err != nil {
		t.Fatal(err)
	}
	if repos.Auth == nil {
		t.Fatal("expected auth repository placeholder")
	}
}
