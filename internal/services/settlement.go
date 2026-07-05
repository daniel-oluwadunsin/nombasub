package services

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"gorm.io/gorm"
)

type SettlementService struct {
	rc            *repositories.Container
	nombaProvider nomba.Provider
	publisher     *queue.Publisher
	cfg           *config.Config
}

const virtualQueuedPayoutPrefix = "queued:"

func NewSettlementService(rc *repositories.Container, nombaProvider nomba.Provider, publisher *queue.Publisher, cfg *config.Config) *SettlementService {
	return &SettlementService{rc: rc, nombaProvider: nombaProvider, publisher: publisher, cfg: cfg}
}

func (s *SettlementService) GetSettlementPayouts(tenantID string, query requests.SettlementPayoutsQuery) (*responses.SettlementPayoutsResponse, error) {
	page, limit := paginationValues(query.Page, query.Limit, 20, 100)
	status := strings.ToLower(strings.TrimSpace(valueOrEmpty(query.Status)))

	var payouts []models.SettlementPayout
	payoutDB := s.rc.DB.Model(&models.SettlementPayout{}).Where("tenant_id = ?", tenantID)
	payoutDB, err := applySettlementPayoutFilters(payoutDB, query)
	if err != nil {
		return nil, err
	}
	if status != "queued" {
		if err := payoutDB.Order("created_at DESC").Find(&payouts).Error; err != nil {
			return nil, responses.InternalServerError(err)
		}
	}

	items := make([]responses.SettlementPayoutListItem, 0, len(payouts)+1)
	for _, payout := range payouts {
		items = append(items, payoutListItem(payout))
	}

	queued, err := s.virtualQueuedPayout(tenantID)
	if err != nil {
		return nil, err
	}
	if queued != nil && (status == "" || status == "queued") {
		items = append([]responses.SettlementPayoutListItem{queued.SettlementPayoutListItem}, items...)
	}

	if query.Search != nil && strings.TrimSpace(*query.Search) != "" {
		items = filterPayoutItems(items, strings.TrimSpace(*query.Search))
	}

	total := len(items)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := int(math.Min(float64(start+limit), float64(total)))
	pagedItems := items[start:end]

	metrics, err := s.settlementPayoutMetrics(tenantID)
	if err != nil {
		return nil, err
	}

	return &responses.SettlementPayoutsResponse{
		Data: pagedItems,
		Meta: responses.PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      int64(total),
			TotalPages: int(math.Ceil(float64(total) / float64(limit))),
		},
		Metrics: metrics,
	}, nil
}

func (s *SettlementService) GetSettlementPayout(tenantID, payoutID string) (*responses.SettlementPayoutDetail, error) {
	if strings.HasPrefix(payoutID, virtualQueuedPayoutPrefix) {
		detail, err := s.virtualQueuedPayout(tenantID)
		if err != nil {
			return nil, err
		}
		if detail == nil {
			return nil, responses.NotFound("Queued settlement payout not found")
		}
		return detail, nil
	}

	var payout models.SettlementPayout
	if err := s.rc.DB.
		Where("tenant_id = ? AND id = ?", tenantID, payoutID).
		Preload("Settlements", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		First(&payout).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "record not found") {
			return nil, responses.NotFound("Settlement payout not found")
		}
		return nil, responses.InternalServerError(err)
	}

	settlements := make([]responses.SettlementResponse, 0, len(payout.Settlements))
	for _, settlement := range payout.Settlements {
		settlements = append(settlements, settlementResponse(settlement))
	}

	return &responses.SettlementPayoutDetail{
		SettlementPayoutListItem: payoutListItem(payout),
		UpdatedAt:                payout.UpdatedAt,
		Settlements:              settlements,
	}, nil
}

// ProcessDueSettlements is the Mon–Fri cron job. It loads all pending settlements
// whose settlement_date has passed, groups them by tenant+currency, and makes one
// wallet transfer per group.
func (s *SettlementService) ProcessDueSettlements() {
	today := time.Now().Format("2006-01-02")

	settlements, err := s.rc.SettlementRepository.FindManyRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"status = ? AND settlement_date <= ?",
			models.SettlementStatusPending,
			today,
		),
	})
	if err != nil {
		log.Printf("settlement cron: failed to load due settlements: %v", err)
		return
	}
	if len(settlements) == 0 {
		return
	}

	type groupKey struct {
		TenantID string
		Currency string
	}
	groups := make(map[groupKey][]models.Settlement)
	for _, s := range settlements {
		k := groupKey{TenantID: s.TenantID, Currency: s.Currency}
		groups[k] = append(groups[k], s)
	}

	for key, batch := range groups {
		if err := s.processBatch(key.TenantID, key.Currency, batch); err != nil {
			log.Printf("settlement cron: failed batch for tenant %s currency %s: %v", key.TenantID, key.Currency, err)
		}
	}
}

func (s *SettlementService) processBatch(tenantID, currency string, settlements []models.Settlement) error {
	tenant, err := s.rc.TenantRepository.FindById(tenantID, nil)
	if err != nil {
		return err
	}
	if tenant == nil || tenant.AccountID == "" {
		return fmt.Errorf("tenant %s has no Nomba account ID", tenantID)
	}

	var total float64
	for _, s := range settlements {
		total += s.Amount
	}

	ref, err := utils.GenerateNumericString(12)
	if err != nil {
		return err
	}

	var payout *models.SettlementPayout
	var transferErr error

	txErr := s.rc.DB.Transaction(func(trx *gorm.DB) error {
		payout, err = s.rc.SettlementPayoutRepository.Create(&models.SettlementPayout{
			TenantID:           tenantID,
			Amount:             total,
			Currency:           currency,
			Reference:          ref,
			Status:             models.SettlementPayoutStatusPending,
			SettlementCount:    len(settlements),
			RecipientAccountId: &tenant.AccountID,
		}, trx)
		return err
	})
	if txErr != nil {
		return txErr
	}

	narration := fmt.Sprintf("Subscription settlement – %d payment(s)", len(settlements))
	resp, transferErr := s.nombaProvider.TransferToNombaAccount(nomba.TransferToAccountRequest{
		Amount:            total,
		ReceiverAccountId: tenant.AccountID,
		MerchantTxRef:     ref,
		SenderName:        s.cfg.NombaSenderName,
		Narration:         narration,
	})

	return s.rc.DB.Transaction(func(trx *gorm.DB) error {
		if transferErr != nil {
			reason := transferErr.Error()
			payout.Status = models.SettlementPayoutStatusFailed
			payout.FailureReason = &reason
			if _, err := s.rc.SettlementPayoutRepository.Update(payout, trx); err != nil {
				return err
			}
			s.enqueueWebhook(tenantID, models.WebhookDeliveryEventTypeSettlementPayoutFailed, map[string]interface{}{
				"payout":      payout,
				"settlements": settlements,
			}, trx)
			return nil
		}

		now := time.Now()
		payout.Status = models.SettlementPayoutStatusCompleted
		payout.ProcessedAt = &now
		if resp.Data.ID != nil {
			payout.NombaTransactionID = resp.Data.ID
		}
		if _, err := s.rc.SettlementPayoutRepository.Update(payout, trx); err != nil {
			return err
		}

		ids := make([]string, len(settlements))
		for i, settlement := range settlements {
			ids[i] = settlement.ID
		}
		if err := trx.Model(&models.Settlement{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":               models.SettlementStatusCompleted,
				"settlement_payout_id": payout.ID,
			}).Error; err != nil {
			return err
		}

		s.enqueueWebhook(tenantID, models.WebhookDeliveryEventTypeSettlementPayoutInitiated, map[string]interface{}{
			"payout":      payout,
			"settlements": settlements,
		}, trx)

		return nil
	})
}

func (s *SettlementService) enqueueWebhook(tenantID string, eventType models.WebhookDeliveryEventType, data any, trx *gorm.DB) {
	if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantID, eventType, data, trx); err != nil {
		log.Printf("settlement webhook enqueue failed for tenant %s event %s: %v", tenantID, eventType, err)
	}
}

func (s *SettlementService) virtualQueuedPayout(tenantID string) (*responses.SettlementPayoutDetail, error) {
	tenant, err := s.rc.TenantRepository.FindById(tenantID, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	var settlements []models.Settlement
	if err := s.rc.DB.
		Where("tenant_id = ? AND status = ? AND settlement_payout_id IS NULL", tenantID, models.SettlementStatusPending).
		Order("created_at DESC").
		Find(&settlements).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}
	if len(settlements) == 0 {
		return nil, nil
	}

	var total float64
	currency := "NGN"
	createdAt := settlements[0].CreatedAt
	settlementResponses := make([]responses.SettlementResponse, 0, len(settlements))
	for _, settlement := range settlements {
		total += settlement.Amount
		currency = settlement.Currency
		if settlement.CreatedAt.Before(createdAt) {
			createdAt = settlement.CreatedAt
		}
		settlementResponses = append(settlementResponses, settlementResponse(settlement))
	}

	var recipient *string
	if tenant != nil && tenant.AccountID != "" {
		recipient = &tenant.AccountID
	}

	return &responses.SettlementPayoutDetail{
		SettlementPayoutListItem: responses.SettlementPayoutListItem{
			ID:                 virtualQueuedPayoutPrefix + currency,
			Amount:             total,
			Currency:           currency,
			Reference:          "queued-settlements",
			Status:             "queued",
			CreatedAt:          createdAt,
			SettlementCount:    len(settlements),
			RecipientAccountID: recipient,
			IsVirtual:          true,
		},
		UpdatedAt:   createdAt,
		Settlements: settlementResponses,
	}, nil
}

func (s *SettlementService) settlementPayoutMetrics(tenantID string) (responses.SettlementPayoutMetrics, error) {
	var paid struct {
		TotalPaidOut    float64
		LastPaidOutDate *time.Time
		Currency        string
	}
	if err := s.rc.DB.Model(&models.SettlementPayout{}).
		Select("COALESCE(SUM(amount), 0) as total_paid_out, MAX(processed_at) as last_paid_out_date, COALESCE(MAX(currency), 'NGN') as currency").
		Where("tenant_id = ? AND status = ?", tenantID, models.SettlementPayoutStatusCompleted).
		Scan(&paid).Error; err != nil {
		return responses.SettlementPayoutMetrics{}, responses.InternalServerError(err)
	}

	var pending struct {
		Amount float64
		Count  int64
	}
	if err := s.rc.DB.Model(&models.Settlement{}).
		Select("COALESCE(SUM(amount), 0) as amount, COUNT(*) as count").
		Where("tenant_id = ? AND status = ? AND settlement_payout_id IS NULL", tenantID, models.SettlementStatusPending).
		Scan(&pending).Error; err != nil {
		return responses.SettlementPayoutMetrics{}, responses.InternalServerError(err)
	}

	currency := paid.Currency
	if currency == "" {
		currency = "NGN"
	}

	return responses.SettlementPayoutMetrics{
		TotalPaidOut:                      paid.TotalPaidOut,
		PendingSettlementAmount:           pending.Amount,
		PendingSettlementTransactionCount: pending.Count,
		LastPaidOutDate:                   paid.LastPaidOutDate,
		Currency:                          currency,
	}, nil
}

func applySettlementPayoutFilters(db *gorm.DB, query requests.SettlementPayoutsQuery) (*gorm.DB, error) {
	status := strings.ToLower(strings.TrimSpace(valueOrEmpty(query.Status)))
	if status != "" && status != "queued" {
		db = db.Where("status = ?", status)
	}
	if query.From != nil && strings.TrimSpace(*query.From) != "" {
		from, err := parseSettlementDate(*query.From, false)
		if err != nil {
			return nil, responses.BadRequest("from must use YYYY-MM-DD format")
		}
		db = db.Where("created_at >= ?", *from)
	}
	if query.To != nil && strings.TrimSpace(*query.To) != "" {
		to, err := parseSettlementDate(*query.To, true)
		if err != nil {
			return nil, responses.BadRequest("to must use YYYY-MM-DD format")
		}
		db = db.Where("created_at <= ?", *to)
	}
	return db, nil
}

func filterPayoutItems(items []responses.SettlementPayoutListItem, search string) []responses.SettlementPayoutListItem {
	search = strings.ToLower(search)
	filtered := make([]responses.SettlementPayoutListItem, 0, len(items))
	for _, item := range items {
		values := []string{item.ID, item.Reference, item.Status}
		if item.NombaTransactionID != nil {
			values = append(values, *item.NombaTransactionID)
		}
		if item.RecipientAccountID != nil {
			values = append(values, *item.RecipientAccountID)
		}
		for _, value := range values {
			if strings.Contains(strings.ToLower(value), search) {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

func payoutListItem(payout models.SettlementPayout) responses.SettlementPayoutListItem {
	return responses.SettlementPayoutListItem{
		ID:                 payout.ID,
		Amount:             payout.Amount,
		Currency:           payout.Currency,
		Reference:          payout.Reference,
		Status:             string(payout.Status),
		NombaTransactionID: payout.NombaTransactionID,
		FailureReason:      payout.FailureReason,
		ProcessedAt:        payout.ProcessedAt,
		CreatedAt:          payout.CreatedAt,
		SettlementCount:    payout.SettlementCount,
		RecipientAccountID: payout.RecipientAccountId,
		IsVirtual:          false,
	}
}

func settlementResponse(settlement models.Settlement) responses.SettlementResponse {
	return responses.SettlementResponse{
		ID:                 settlement.ID,
		Amount:             settlement.Amount,
		Currency:           settlement.Currency,
		SettlementDate:     settlement.SettlementTime,
		Status:             string(settlement.Status),
		Reference:          settlement.Reference,
		FailureReason:      settlement.FailureReason,
		Purpose:            string(settlement.Purpose),
		SubscriptionID:     settlement.SubscriptionID,
		InvoiceID:          settlement.InvoiceID,
		SettlementPayoutID: settlement.SettlementPayoutID,
		CreatedAt:          settlement.CreatedAt,
		UpdatedAt:          settlement.UpdatedAt,
	}
}

func paginationValues(pagePtr, limitPtr *int, defaultLimit, maxLimit int) (int, int) {
	page := 1
	limit := defaultLimit
	if pagePtr != nil && *pagePtr > 0 {
		page = *pagePtr
	}
	if limitPtr != nil && *limitPtr > 0 {
		limit = *limitPtr
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

func parseSettlementDate(value string, endOfDay bool) (*time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
