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
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 5)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ValidateHash(hash string, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) != nil
}

func GenerateJwt(tenantId string, cfg *config.Config) (string, error) {
	jwtSecret := cfg.JWTSecret

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"tenantId": tenantId, "timestamp": time.Now().String()})

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

	claims := token.Claims.(jwt.MapClaims)

	tenantId := claims["tenantId"].(string)

	return tenantId, nil
}
