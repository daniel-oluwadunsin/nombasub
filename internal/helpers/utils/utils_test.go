package utils

import (
	"testing"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateJwtRoundTrip(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-value-please-ignore"}

	token, err := GenerateJwt("tenant-123", cfg)
	if err != nil {
		t.Fatalf("GenerateJwt: %v", err)
	}

	tenantID, err := ValidateJwt(token, cfg)
	if err != nil {
		t.Fatalf("ValidateJwt: %v", err)
	}
	if tenantID != "tenant-123" {
		t.Fatalf("tenantID = %q, want tenant-123", tenantID)
	}
}

func TestValidateJwtRejectsExpired(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-value-please-ignore"}

	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenantId": "tenant-123",
		"exp":      time.Now().Add(-time.Hour).Unix(),
	})
	signed, err := expired.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if _, err := ValidateJwt(signed, cfg); err == nil {
		t.Fatal("expected expired token to be rejected, got nil error")
	}
}

func TestValidateJwtRejectsWrongSecret(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-value-please-ignore"}
	other := &config.Config{JWTSecret: "a-completely-different-secret-val"}

	token, err := GenerateJwt("tenant-123", cfg)
	if err != nil {
		t.Fatalf("GenerateJwt: %v", err)
	}
	if _, err := ValidateJwt(token, other); err == nil {
		t.Fatal("expected token signed with a different secret to be rejected")
	}
}

func TestHashAPIKeyIsDeterministicAndDistinct(t *testing.T) {
	a := HashAPIKey("key-one")
	if a != HashAPIKey("key-one") {
		t.Fatal("HashAPIKey is not deterministic")
	}
	if a == HashAPIKey("key-two") {
		t.Fatal("different keys produced the same hash")
	}
}

func TestMaskSecretDoesNotRevealFullValue(t *testing.T) {
	secret := "abcdefghijklmnopqrstuvwxyz"
	masked := MaskSecret(secret)
	if masked == secret {
		t.Fatal("MaskSecret returned the full secret")
	}
	if len(masked) > 10 {
		t.Fatalf("MaskSecret revealed too much: %q", masked)
	}
}
