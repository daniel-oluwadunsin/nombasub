package services

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

type WebhookService struct {
	rc          *repositories.Container
	nombaClient nomba.Provider
}

func NewWebhookService(rc *repositories.Container, nombaClient nomba.Provider) *WebhookService {
	return &WebhookService{rc: rc, nombaClient: nombaClient}
}

func (ws WebhookService) convertRequestBodyToJson(body interface{}) (*string, error) {
	var jsonData string

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	jsonData = string(jsonBytes)
	return &jsonData, nil
}

func (ws *WebhookService) ValidateWebhookSignature(receivedSignature, timestamp string, payload nomba.NombaWebhookRequest) bool {
	payloadJson, err := ws.convertRequestBodyToJson(payload)
	if err != nil {
		return false
	}

	expectedSignature, err :=
		ws.nombaClient.GenerateSignature(*payloadJson, timestamp)
	if err != nil {
		return false
	}

	return receivedSignature == expectedSignature
}

func (ws *WebhookService) updateNombaInitiationStatus(initiationId string, status models.NombaInitiationStatus) error {
	initiationRepository := ws.rc.NombaInitiationRepository

	_, err := initiationRepository.Update(&models.NombaInitiation{BaseModel: models.BaseModel{ID: initiationId}}, nil)
	return err
}

func (ws *WebhookService) handlePaymentSuccess(payload nomba.NombaWebhookRequest) error {
	initiationRepository := ws.rc.NombaInitiationRepository
	customerRepository := ws.rc.CustomerRepository
	planVersionRepository := ws.rc.PlanVersionRepository
	paymentSourceRepository := ws.rc.PaymentSourceRepository
	settlementRepository := ws.rc.SettlementRepository
	nombaClient := ws.nombaClient

	if payload.Data.Transaction.Type == nomba.TransactionTypeOnlineCheckout {
		orderId := payload.Data.Order.OrderId
		orderReference := utils.OrStrings(payload.Data.Order.OrderReference, payload.Data.Transaction.MerchantTxRef)

		initiation, err := initiationRepository.FindRaw(&repositories.FindArgs{
			Filter: repositories.NewQueryFilter().Where(
				"reference = ? OR nomba_order_id = ?", orderReference, orderId,
			),
		})

		if err != nil {
			return err
		}

		if initiation == nil {
			return nil
		}

		if initiation.Purpose == models.NombaInitiationPurposeCardSubscriptionPayment {
			tokenizedCard := payload.Data.TokenizedCardData
			if tokenizedCard == nil {
				return errors.New("tokenized card data is missing in the webhook payload")
			}
			tenantId := initiation.Metadata["nombaSubTenantId"].(string)
			customerCode := initiation.Metadata["nombaSubCustomerCode"].(string)
			planCode := initiation.Metadata["nombaSubPlanCode"].(string)
			planVersionNumber := initiation.Metadata["nombaSubPlanVersion"].(int)

			customer, err := customerRepository.Find(&models.Customer{TenantID: tenantId, Code: customerCode}, nil)

			if err != nil {
				return err
			}

			planVersion, err := planVersionRepository.Find(
				&models.PlanVersion{
					TenantID: tenantId,
					Code:     planCode,
					Index:    planVersionNumber,
				},
				nil,
			)
			if err != nil {
				return err
			}

			if customer == nil || planVersion == nil {
				return errors.New("customer or plan not found for the given tenant and codes")
			}

			card, err := paymentSourceRepository.Find(&models.PaymentSource{
				TenantID:   tenantId,
				CustomerID: customer.ID,
				Type:       models.PaymentSourceTypeCard,
				Card: &models.CardPaymentSource{
					AuthorizationToken: &tokenizedCard.TokenKey,
				},
			}, nil)

			if err != nil {
				return err
			}

			if card == nil {
				card, err = paymentSourceRepository.Create(&models.PaymentSource{
					TenantID:   tenantId,
					CustomerID: customer.ID,
					Type:       models.PaymentSourceTypeCard,
					Card: &models.CardPaymentSource{
						Type:               utils.OrStrings(tokenizedCard.CardType, payload.Data.Order.CardType),
						Pan:                &tokenizedCard.CardPan,
						Last4Digits:        &payload.Data.Order.CardLast4Digits,
						Currency:           &payload.Data.Order.CardCurrency,
						AuthorizationToken: &tokenizedCard.TokenKey,
					},
					Status: models.PaymentSourceStatusActive,
				}, nil)

				if err != nil {
					return err
				}
			}

			subscription := &models.Subscription{
				TenantID:          tenantId,
				CustomerID:        customer.ID,
				PlanID:            planVersion.PlanID,
				PlanVersionID:     planVersion.ID,
				PaymentSourceID:   card.ID,
				PaymentSourceType: models.PaymentSourceTypeCard,
				Interval:          planVersion.Interval,
				Amount:            planVersion.Amount,
				IntervalCount:     planVersion.IntervalCount,
				TrialPeriodDays:   planVersion.TrialPeriodDays,
				Currency:          planVersion.Currency,
				InvoiceLimit:      planVersion.InvoiceLimit,
			}

			if planVersion.TrialPeriodDays != 0 {
				subscription.TrialStartDate = utils.ToPtr(time.Now())
				subscription.TrialEndDate = utils.ToPtr(time.Now().AddDate(0, 0, planVersion.TrialPeriodDays))

				startDate, endDate := utils.GetBillingPeriod(*subscription.TrialEndDate, planVersion.Interval, planVersion.IntervalCount)
				subscription.CurrentBillingCycleStart = &startDate
				subscription.CurrentBillingCycleEnd = &endDate
			}

			if planVersion.TrialPeriodDays == 0 {
				startDate, endDate := utils.GetBillingPeriod(time.Now(), planVersion.Interval, planVersion.IntervalCount)
				subscription.CurrentBillingCycleStart = &startDate
				subscription.CurrentBillingCycleEnd = &endDate
				subscription.StartedAt = utils.ToPtr(time.Now())
			}

			_, err = ws.rc.SubscriptionRepository.Create(subscription, nil)
			if err != nil {
				return err
			}

			amountAfterFee := nombaClient.DeductFee(float64(planVersion.Amount) / 100)

			_, err = settlementRepository.Create(&models.Settlement{
				TenantID:       tenantId,
				Purpose:        initiation.Purpose,
				Amount:         amountAfterFee,
				Currency:       planVersion.Currency,
				Status:         models.SettlementStatusPending,
				Reference:      initiation.Reference,
				SettlementTime: time.Now().Add(25 * time.Hour),
			}, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ws *WebhookService) handlePaymentFailed(payload nomba.NombaWebhookRequest) error {
	// Implement the logic to handle payment failed webhook event
	return nil
}

func (ws *WebhookService) handlePaymentReversal(payload nomba.NombaWebhookRequest) error {
	// Implement the logic to handle payment reversal webhook event
	return nil
}

func (ws *WebhookService) handlePayoutSuccess(payload nomba.NombaWebhookRequest) error {
	// Implement the logic to handle payout success webhook event
	return nil
}

func (ws *WebhookService) handlePayoutFailed(payload nomba.NombaWebhookRequest) error {
	// Implement the logic to handle payout failed webhook event
	return nil
}

func (ws *WebhookService) handlePayoutRefund(payload nomba.NombaWebhookRequest) error {
	// Implement the logic to handle payout refund webhook event
	return nil
}

func (ws *WebhookService) HandleWebhook(payload nomba.NombaWebhookRequest) error {
	eventType := payload.EventType

	switch eventType {
	case nomba.WebhookEventTypePaymentSuccess:
		return ws.handlePaymentSuccess(payload)
	case nomba.WebhookEventTypePaymentFailed:
		return ws.handlePaymentFailed(payload)
	case nomba.WebhookEventTypePaymentReversal:
		return ws.handlePaymentReversal(payload)
	case nomba.WebhookEventTypePayoutSuccess:
		return ws.handlePayoutSuccess(payload)
	case nomba.WebhookEventTypePayoutFailed:
		return ws.handlePayoutFailed(payload)
	case nomba.WebhookEventTypePayoutRefund:
		return ws.handlePayoutRefund(payload)
	default:
		return nil
	}
}
