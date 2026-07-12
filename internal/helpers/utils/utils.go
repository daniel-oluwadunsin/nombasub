package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func GenerateNumericString(length int) (string, error) {
	const charset = "0123456789"
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	for i, b := range randomBytes {
		randomBytes[i] = charset[b%byte(len(charset))]
	}
	return string(randomBytes), nil
}

func GenerateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	for i, b := range randomBytes {
		randomBytes[i] = charset[b%byte(len(charset))]
	}

	return string(randomBytes), nil
}

func GenerateCode(prefix string) (string, error) {
	defaultCodeLength := 8
	randomString, err := GenerateRandomString(defaultCodeLength)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s_%s", prefix, randomString), nil
}

func Or[T any](value *T, defaultValue *T) *T {
	if value != nil {
		return value
	}
	return defaultValue
}

func OrStrings(value ...string) string {
	for _, v := range value {
		if v != "" {
			return v
		}
	}
	return ""
}

func ToPtr[T any](value T) *T {
	return &value
}

func GetBillingPeriod(startDate time.Time, interval models.PlanInterval, billingIntervalCount *int) (time.Time, time.Time) {
	intervalCount := 1
	if billingIntervalCount != nil && *billingIntervalCount > 0 {
		intervalCount = *billingIntervalCount
	}

	switch interval {
	case models.PlanIntervalDaily:
		return startDate, startDate.AddDate(0, 0, intervalCount)
	case models.PlanIntervalWeekly:
		return startDate, startDate.AddDate(0, 0, 7*intervalCount)
	case models.PlanIntervalBiWeekly:
		return startDate, startDate.AddDate(0, 0, 14*intervalCount)
	case models.PlanIntervalMonthly:
		return startDate, startDate.AddDate(0, intervalCount, 0)
	case models.PlanIntervalQuarterly:
		return startDate, startDate.AddDate(0, 3*intervalCount, 0)
	case models.PlanIntervalYearly:
		return startDate, startDate.AddDate(intervalCount, 0, 0)
	default:
		return startDate, startDate
	}
}

func HashOutgoingPayload(eventType, webhookId, tenantId, timestamp string, webhookSecret string) string {
	hashingPayload := fmt.Sprintf(
		"%s:%s:%s:%s",
		eventType,
		webhookId,
		tenantId,
		timestamp,
	)

	h := hmac.New(sha256.New, []byte(webhookSecret))
	h.Write([]byte(hashingPayload))
	hash := h.Sum(nil)

	return base64.StdEncoding.EncodeToString(hash)
}

func Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ValidateHash(hash string, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func DigestToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// HashAPIKey returns a deterministic digest for an API key. It is deterministic
// (unlike bcrypt) so the value can be indexed and looked up directly, while the
// plaintext key never has to be stored. Safe because API keys are high-entropy.
func HashAPIKey(key string) string {
	return DigestToken(key)
}

// MaskSecret returns a non-reversible preview of a secret for display purposes
// (e.g. showing which key is configured without revealing it).
func MaskSecret(secret string) string {
	if len(secret) <= 6 {
		return "…"
	}
	return secret[:6] + "…"
}

// TenantTokenTTL is the lifetime of a tenant access token. It must match the
// AccessTokenExpiresAt persisted at login so the cryptographic expiry and the
// database revocation gate agree.
const TenantTokenTTL = 24 * time.Hour

func GenerateJwt(tenantId string, cfg *config.Config) (string, error) {
	jwtSecret := cfg.JWTSecret

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenantId": tenantId,
		"iat":      now.Unix(),
		"exp":      now.Add(TenantTokenTTL).Unix(),
	})

	return token.SignedString([]byte(jwtSecret))
}

func ValidateJwt(tokenString string, cfg *config.Config) (string, error) {
	jwtSecret := cfg.JWTSecret

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}

		return []byte(jwtSecret), nil
	})

	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", jwt.ErrTokenInvalidClaims
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	tenantId, ok := claims["tenantId"].(string)
	if !ok || tenantId == "" {
		return "", jwt.ErrTokenInvalidClaims
	}

	return tenantId, nil
}

type PortalJwtClaims struct {
	TenantID   string
	CustomerID string
	SessionID  string
}

func GeneratePortalJwt(tenantID, customerID, sessionID string, expiresAt time.Time, cfg *config.Config) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenantId":   tenantID,
		"customerId": customerID,
		"sessionId":  sessionID,
		"aud":        "customer_portal",
		"exp":        expiresAt.Unix(),
		"iat":        time.Now().Unix(),
	})

	return token.SignedString([]byte(cfg.JWTSecret))
}

func ValidatePortalJwt(tokenString string, cfg *config.Config) (*PortalJwtClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	if claims["aud"] != "customer_portal" {
		return nil, jwt.ErrTokenInvalidAudience
	}

	tenantID, tenantOK := claims["tenantId"].(string)
	customerID, customerOK := claims["customerId"].(string)
	sessionID, sessionOK := claims["sessionId"].(string)
	if !tenantOK || !customerOK || !sessionOK {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return &PortalJwtClaims{
		TenantID:   tenantID,
		CustomerID: customerID,
		SessionID:  sessionID,
	}, nil
}
