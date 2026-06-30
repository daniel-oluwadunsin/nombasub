package utils

import (
	"crypto/rand"
	"fmt"
)

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

func ToPtr[T any](value T) *T {
	return &value
}
