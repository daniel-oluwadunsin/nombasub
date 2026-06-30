package nomba

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
)

func (c *Client) GenerateSignature(payloadJSON, timeStamp string) (string, error) {
	clientSecret := c.WebhookSecret

	var payload NombaWebhookRequest
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return "", fmt.Errorf("error parsing JSON payload: %w", err)
	}

	transaction := payload.Data.Transaction
	merchant := payload.Data.Merchant

	transactionResponseCode := transaction.ResponseCode
	if transactionResponseCode == "null" {
		transactionResponseCode = ""
	}

	// Construct the exact signature payload as in Java
	hashingPayload := fmt.Sprintf(
		"%s:%s:%s:%s:%s:%s:%s:%s:%s",
		payload.EventType,
		payload.RequestID,
		merchant.UserID,
		merchant.WalletID,
		transaction.TransactionID,
		transaction.Type,
		transaction.Time,
		transactionResponseCode,
		timeStamp,
	)

	log.Printf("::: payload to hash --> [%s] :::", hashingPayload)

	// Generate HMAC SHA256 and encode Base64
	h := hmac.New(sha256.New, []byte(clientSecret))
	h.Write([]byte(hashingPayload))
	hash := h.Sum(nil)

	return base64.StdEncoding.EncodeToString(hash), nil
}

func (c *Client) CalculateFee(amount float64) float64 {
	var feeCap float64 = 1800
	feeAmount := 1.4 / 100 * amount

	return min(feeAmount, feeCap)
}

func (c *Client) DeductFee(amount float64) float64 {
	fee := c.CalculateFee(amount)
	return amount - fee
}
