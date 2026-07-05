package responses

import "time"

type WebhookDeliveryListResponse struct {
	Data  []WebhookDeliveryListItem `json:"data"`
	Meta  PaginationMeta            `json:"meta"`
	Stats WebhookDeliveryStats      `json:"stats"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

type WebhookDeliveryStats struct {
	Total      int64 `json:"total"`
	Successful int64 `json:"successful"`
	Failed     int64 `json:"failed"`
	Retrying   int64 `json:"retrying"`
}

type WebhookDeliveryListItem struct {
	ID              string                  `json:"id"`
	EndpointURL     string                  `json:"endpointUrl"`
	EventType       string                  `json:"eventType"`
	Payload         string                  `json:"payload"`
	Status          string                  `json:"status"`
	RetryCount      int                     `json:"retryCount"`
	MaxAttemptCount int                     `json:"maxAttemptCount"`
	CreatedAt       time.Time               `json:"createdAt"`
	UpdatedAt       time.Time               `json:"updatedAt"`
	LatestAttempt   *WebhookAttemptResponse `json:"latestAttempt"`
}

type WebhookDeliveryDetail struct {
	ID              string                   `json:"id"`
	EndpointURL     string                   `json:"endpointUrl"`
	EventType       string                   `json:"eventType"`
	Payload         string                   `json:"payload"`
	Status          string                   `json:"status"`
	RetryCount      int                      `json:"retryCount"`
	MaxAttemptCount int                      `json:"maxAttemptCount"`
	CreatedAt       time.Time                `json:"createdAt"`
	UpdatedAt       time.Time                `json:"updatedAt"`
	Attempts        []WebhookAttemptResponse `json:"attempts"`
}

type WebhookAttemptResponse struct {
	ID                string    `json:"id"`
	WebhookDeliveryID string    `json:"webhookDeliveryId"`
	StatusCode        int       `json:"statusCode"`
	ResponseBody      string    `json:"responseBody"`
	AttemptCount      int       `json:"attemptCount"`
	CreatedAt         time.Time `json:"createdAt"`
}
