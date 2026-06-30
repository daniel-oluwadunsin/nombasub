package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
)

type EncryptedSecret struct {
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	Algorithm  string `json:"algorithm"`
	KeyVersion string `json:"keyVersion"`
}

const Algorithm = "AES-256-GCM"

func Encrypt(value string, identifier string) (*EncryptedSecret, error) {
	config := config.Load()
	base64Key := config.EncryptionKey
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, err
	}

	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	var additionalData []byte
	if identifier != "" {
		additionalData = []byte(identifier)
	}

	ciphertext := gcm.Seal(
		nil,
		nonce,
		[]byte(value),
		additionalData,
	)

	return &EncryptedSecret{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Algorithm:  Algorithm,
		KeyVersion: "v1",
	}, nil
}

func Decrypt(payload EncryptedSecret, identifier string) (string, error) {
	config := config.Load()
	base64Key := config.EncryptionKey

	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return "", err
	}

	if len(key) != 32 {
		return "", errors.New("encryption key must be 32 bytes for AES-256")
	}

	if payload.Algorithm != Algorithm {
		return "", errors.New("unsupported encryption algorithm")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return "", err
	}

	var additionalData []byte
	if identifier != "" {
		additionalData = []byte(identifier)
	}

	plaintext, err := gcm.Open(
		nil,
		nonce,
		ciphertext,
		additionalData,
	)
	if err != nil {
		return "", errors.New("failed to decrypt client secret")
	}

	return string(plaintext), nil
}
