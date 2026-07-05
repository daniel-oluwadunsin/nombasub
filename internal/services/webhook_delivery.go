package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"gorm.io/gorm"
)

type WebhookDeliveryService struct {
	rc        *repositories.Container
	publisher *queue.Publisher
}

func NewWebhookDeliveryService(rc *repositories.Container, publisher *queue.Publisher) *WebhookDeliveryService {
	return &WebhookDeliveryService{rc: rc, publisher: publisher}
}

func (s *WebhookDeliveryService) ListWebhookDeliveries(tenantID string, query requests.WebhookDeliveriesQuery) (*responses.WebhookDeliveryListResponse, error) {
	page := 1
	limit := 20
	if query.Page != nil && *query.Page > 0 {
		page = *query.Page
	}
	if query.Limit != nil && *query.Limit > 0 {
		limit = *query.Limit
	}
	if limit > 100 {
		limit = 100
	}

	db := s.rc.DB.Model(&models.WebhookDelivery{}).Where("tenant_id = ?", tenantID)
	db, err := applyWebhookDeliveryFilters(db, query)
	if err != nil {
		return nil, err
	}
	stats, err := s.webhookDeliveryStats(tenantID, query)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var deliveries []models.WebhookDelivery
	if err := db.
		Order("created_at DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&deliveries).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	latestAttempts, err := s.latestAttemptsForDeliveries(deliveries)
	if err != nil {
		return nil, err
	}

	items := make([]responses.WebhookDeliveryListItem, 0, len(deliveries))
	for _, delivery := range deliveries {
		items = append(items, deliveryListItem(delivery, latestAttempts[delivery.ID]))
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return &responses.WebhookDeliveryListResponse{
		Data: items,
		Meta: responses.PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
		Stats: stats,
	}, nil
}

func (s *WebhookDeliveryService) GetWebhookDelivery(tenantID, deliveryID string) (*responses.WebhookDeliveryDetail, error) {
	delivery, err := s.findTenantDelivery(tenantID, deliveryID)
	if err != nil {
		return nil, err
	}

	attempts, err := s.deliveryAttempts(delivery.ID)
	if err != nil {
		return nil, err
	}

	return deliveryDetail(*delivery, attempts), nil
}

func (s *WebhookDeliveryService) RetryWebhookDelivery(tenantID, deliveryID string) (*responses.WebhookDeliveryDetail, string, error) {
	delivery, err := s.findTenantDelivery(tenantID, deliveryID)
	if err != nil {
		return nil, "", err
	}

	if delivery.Status == models.WebhookDeliveryStatusDelivered {
		detail, err := s.GetWebhookDelivery(tenantID, deliveryID)
		return detail, "Webhook delivery already succeeded.", err
	}

	if delivery.AttempsCount >= delivery.MaxAttemptCount {
		delivery.MaxAttemptCount = delivery.AttempsCount + 1
	}
	delivery.Status = models.WebhookDeliveryStatusPending

	if _, err := s.rc.WebhookDeliveryRepository.Update(delivery, nil); err != nil {
		return nil, "", responses.InternalServerError(err)
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(queue.SendTenantWebhookQueue, queue.SendTenantWebhookJob{WebhookDeliveryID: delivery.ID}); err != nil {
			return nil, "", responses.InternalServerError(err)
		}
		detail, err := s.GetWebhookDelivery(tenantID, deliveryID)
		return detail, "Webhook retry queued successfully.", err
	}

	delivery.AttempsCount++
	delivery.Status = models.WebhookDeliveryStatusFailed
	if _, err := s.rc.WebhookDeliveryAttemptRepository.Create(&models.WebhookDeliveryAttempt{
		WebhookDeliveryID: delivery.ID,
		StatusCode:        0,
		ResponseBody:      "Retry requested, but webhook queue is unavailable in this runtime.",
		AttemptCount:      delivery.AttempsCount,
	}, nil); err != nil {
		return nil, "", responses.InternalServerError(err)
	}
	if _, err := s.rc.WebhookDeliveryRepository.Update(delivery, nil); err != nil {
		return nil, "", responses.InternalServerError(err)
	}

	detail, err := s.GetWebhookDelivery(tenantID, deliveryID)
	return detail, "Webhook retry recorded, but queue is unavailable.", err
}

func (s *WebhookDeliveryService) findTenantDelivery(tenantID, deliveryID string) (*models.WebhookDelivery, error) {
	var delivery models.WebhookDelivery
	result := s.rc.DB.
		Where("tenant_id = ? AND id = ?", tenantID, deliveryID).
		First(&delivery)
	if result.Error != nil {
		if strings.Contains(strings.ToLower(result.Error.Error()), "record not found") {
			return nil, responses.NotFound("Webhook delivery not found")
		}
		return nil, responses.InternalServerError(result.Error)
	}
	return &delivery, nil
}

func (s *WebhookDeliveryService) latestAttemptsForDeliveries(deliveries []models.WebhookDelivery) (map[string]*models.WebhookDeliveryAttempt, error) {
	attemptsByDelivery := map[string]*models.WebhookDeliveryAttempt{}
	if len(deliveries) == 0 {
		return attemptsByDelivery, nil
	}

	ids := make([]string, 0, len(deliveries))
	for _, delivery := range deliveries {
		ids = append(ids, delivery.ID)
	}

	var attempts []models.WebhookDeliveryAttempt
	if err := s.rc.DB.
		Where("webhook_delivery_id IN ?", ids).
		Order("webhook_delivery_id ASC, attempt_count DESC, created_at DESC").
		Find(&attempts).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	for _, attempt := range attempts {
		attempt := attempt
		if attemptsByDelivery[attempt.WebhookDeliveryID] == nil {
			attemptsByDelivery[attempt.WebhookDeliveryID] = &attempt
		}
	}

	return attemptsByDelivery, nil
}

func (s *WebhookDeliveryService) deliveryAttempts(deliveryID string) ([]models.WebhookDeliveryAttempt, error) {
	var attempts []models.WebhookDeliveryAttempt
	if err := s.rc.DB.
		Where("webhook_delivery_id = ?", deliveryID).
		Order("attempt_count ASC, created_at ASC").
		Find(&attempts).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}
	return attempts, nil
}

func (s *WebhookDeliveryService) webhookDeliveryStats(tenantID string, query requests.WebhookDeliveriesQuery) (responses.WebhookDeliveryStats, error) {
	db := s.rc.DB.Model(&models.WebhookDelivery{}).Where("tenant_id = ?", tenantID)
	queryWithoutStatus := query
	queryWithoutStatus.Status = nil
	db, err := applyWebhookDeliveryFilters(db, queryWithoutStatus)
	if err != nil {
		return responses.WebhookDeliveryStats{}, err
	}

	var rows []struct {
		Status string
		Count  int64
	}
	if err := db.Select("status, count(*) as count").Group("status").Scan(&rows).Error; err != nil {
		return responses.WebhookDeliveryStats{}, responses.InternalServerError(err)
	}

	var stats responses.WebhookDeliveryStats
	for _, row := range rows {
		stats.Total += row.Count
		switch row.Status {
		case string(models.WebhookDeliveryStatusDelivered):
			stats.Successful += row.Count
		case string(models.WebhookDeliveryStatusFailed):
			stats.Failed += row.Count
		case string(models.WebhookDeliveryStatusPending):
			stats.Retrying += row.Count
		}
	}
	return stats, nil
}

func applyWebhookDeliveryFilters(db *gorm.DB, query requests.WebhookDeliveriesQuery) (*gorm.DB, error) {
	if query.Status != nil && strings.TrimSpace(*query.Status) != "" {
		db = db.Where("status = ?", webhookDeliveryStatusFromQuery(*query.Status))
	}
	if query.EventType != nil && strings.TrimSpace(*query.EventType) != "" {
		db = db.Where("event_type = ?", strings.TrimSpace(*query.EventType))
	}
	if query.Search != nil && strings.TrimSpace(*query.Search) != "" {
		search := "%" + strings.TrimSpace(*query.Search) + "%"
		db = db.Where("id ILIKE ? OR event_type ILIKE ? OR endpoint_url ILIKE ?", search, search, search)
	}
	if query.From != nil {
		from, err := parseWebhookDate(*query.From, false)
		if err != nil {
			return nil, responses.BadRequest("from must use YYYY-MM-DD format")
		}
		if from != nil {
			db = db.Where("created_at >= ?", *from)
		}
	}
	if query.To != nil {
		to, err := parseWebhookDate(*query.To, true)
		if err != nil {
			return nil, responses.BadRequest("to must use YYYY-MM-DD format")
		}
		if to != nil {
			db = db.Where("created_at <= ?", *to)
		}
	}
	return db, nil
}

func deliveryListItem(delivery models.WebhookDelivery, latestAttempt *models.WebhookDeliveryAttempt) responses.WebhookDeliveryListItem {
	return responses.WebhookDeliveryListItem{
		ID:              delivery.ID,
		EndpointURL:     delivery.EndpointURL,
		EventType:       string(delivery.EventType),
		Payload:         delivery.Payload,
		Status:          normalizeWebhookDeliveryStatus(delivery.Status),
		RetryCount:      delivery.AttempsCount,
		MaxAttemptCount: delivery.MaxAttemptCount,
		CreatedAt:       delivery.CreatedAt,
		UpdatedAt:       delivery.UpdatedAt,
		LatestAttempt:   attemptResponsePtr(latestAttempt),
	}
}

func deliveryDetail(delivery models.WebhookDelivery, attempts []models.WebhookDeliveryAttempt) *responses.WebhookDeliveryDetail {
	attemptResponses := make([]responses.WebhookAttemptResponse, 0, len(attempts))
	for _, attempt := range attempts {
		attemptResponses = append(attemptResponses, attemptResponse(attempt))
	}

	return &responses.WebhookDeliveryDetail{
		ID:              delivery.ID,
		EndpointURL:     delivery.EndpointURL,
		EventType:       string(delivery.EventType),
		Payload:         delivery.Payload,
		Status:          normalizeWebhookDeliveryStatus(delivery.Status),
		RetryCount:      delivery.AttempsCount,
		MaxAttemptCount: delivery.MaxAttemptCount,
		CreatedAt:       delivery.CreatedAt,
		UpdatedAt:       delivery.UpdatedAt,
		Attempts:        attemptResponses,
	}
}

func attemptResponsePtr(attempt *models.WebhookDeliveryAttempt) *responses.WebhookAttemptResponse {
	if attempt == nil {
		return nil
	}
	response := attemptResponse(*attempt)
	return &response
}

func attemptResponse(attempt models.WebhookDeliveryAttempt) responses.WebhookAttemptResponse {
	return responses.WebhookAttemptResponse{
		ID:                attempt.ID,
		WebhookDeliveryID: attempt.WebhookDeliveryID,
		StatusCode:        attempt.StatusCode,
		ResponseBody:      attempt.ResponseBody,
		AttemptCount:      attempt.AttemptCount,
		CreatedAt:         attempt.CreatedAt,
	}
}

func normalizeWebhookDeliveryStatus(status models.WebhookDeliveryStatus) string {
	switch status {
	case models.WebhookDeliveryStatusDelivered:
		return "success"
	case models.WebhookDeliveryStatusFailed:
		return "failed"
	case models.WebhookDeliveryStatusPending:
		return "pending"
	default:
		return strings.ToLower(string(status))
	}
}

func webhookDeliveryStatusFromQuery(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success", "delivered":
		return string(models.WebhookDeliveryStatusDelivered)
	case "failed":
		return string(models.WebhookDeliveryStatusFailed)
	case "pending", "retrying", "processing":
		return string(models.WebhookDeliveryStatusPending)
	default:
		return strings.ToUpper(status)
	}
}

func parseWebhookDate(value string, endOfDay bool) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("date must use YYYY-MM-DD format")
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}
