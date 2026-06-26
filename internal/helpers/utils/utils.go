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

func GenerateCode(prefix string, length int) (string, error) {
	randomString, err := GenerateRandomString(length)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s_%s", prefix, randomString), nil
}
