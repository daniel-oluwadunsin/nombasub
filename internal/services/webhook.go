package services

import "github.com/daniel-oluwadunsin/nombasub/internal/repositories"

type WebhookService struct {
	rc *repositories.Container
}

func NewWebhookService(rc *repositories.Container) *WebhookService {
	return &WebhookService{rc: rc}
}
