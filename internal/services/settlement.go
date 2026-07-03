package services

import (
	"fmt"
	"log"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"gorm.io/gorm"
)

type SettlementService struct {
	rc            *repositories.Container
	nombaProvider nomba.Provider
	publisher     *queue.Publisher
	cfg           *config.Config
}

func NewSettlementService(rc *repositories.Container, nombaProvider nomba.Provider, publisher *queue.Publisher, cfg *config.Config) *SettlementService {
	return &SettlementService{rc: rc, nombaProvider: nombaProvider, publisher: publisher, cfg: cfg}
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
			TenantID:        tenantID,
			Amount:          total,
			Currency:        currency,
			Reference:       ref,
			Status:          models.SettlementPayoutStatusPending,
			SettlementCount: len(settlements),
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
			})
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
		})

		return nil
	})
}

func (s *SettlementService) enqueueWebhook(tenantID string, eventType models.WebhookDeliveryEventType, data any) {
	if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantID, eventType, data); err != nil {
		log.Printf("settlement webhook enqueue failed for tenant %s event %s: %v", tenantID, eventType, err)
	}
}
