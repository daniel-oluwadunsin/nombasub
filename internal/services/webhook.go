package services

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"gorm.io/gorm"
)

type WebhookService struct {
	rc          *repositories.Container
	nombaClient nomba.Provider
	publisher   *queue.Publisher
}

func NewWebhookService(rc *repositories.Container, nombaClient nomba.Provider, publisher *queue.Publisher) *WebhookService {
	return &WebhookService{rc: rc, nombaClient: nombaClient, publisher: publisher}
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

func (ws *WebhookService) handlePaymentSuccess(payload nomba.NombaWebhookRequest) error {
	initiationRepository := ws.rc.NombaInitiationRepository
	customerRepository := ws.rc.CustomerRepository
	planVersionRepository := ws.rc.PlanVersionRepository
	paymentSourceRepository := ws.rc.PaymentSourceRepository
	settlementRepository := ws.rc.SettlementRepository
	nombaClient := ws.nombaClient
	db := ws.rc.DB

	err := db.Transaction(func(trx *gorm.DB) error {
		if payload.Data.Transaction.Type == nomba.TransactionTypeOnlineCheckout {
			orderId := payload.Data.Order.OrderId
			orderReference := utils.OrStrings(payload.Data.Order.OrderReference, payload.Data.Transaction.MerchantTxRef)

			initiation, err := initiationRepository.FindRaw(&repositories.FindArgs{
				Filter: repositories.NewQueryFilter().Where(
					"reference = ? OR nomba_order_id = ?", orderReference, orderId,
				),
				Trx: trx,
			})

			if err != nil {
				return err
			}

			if initiation == nil {
				return nil
			}

			if initiation.Status != models.NombaInitiationStatusPending {
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
				planVersionNumber := int(initiation.Metadata["nombaSubPlanVersion"].(float64))
				tenantOrderReference, _ := initiation.Metadata["nombaSubTenantOrderReference"].(string)
				invoiceId, _ := initiation.Metadata["nombaSubInvoiceId"].(string)
				subscriptionId, _ := initiation.Metadata["nombaSubSubscriptionId"].(string)

				customer, err := customerRepository.Find(&models.Customer{TenantID: tenantId, Code: customerCode}, &repositories.FindArgs{Trx: trx})

				if err != nil {
					return err
				}

				planVersion, err := planVersionRepository.Find(
					&models.PlanVersion{
						TenantID: tenantId,
						Code:     planCode,
						Index:    planVersionNumber,
					},
					&repositories.FindArgs{Trx: trx},
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
				}, &repositories.FindArgs{Trx: trx})

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
							ExpiryMonth:        &tokenizedCard.TokenExpiryMonth,
							ExpiryYear:         &tokenizedCard.TokenExpiryYear,
						},
						Status: models.PaymentSourceStatusActive,
					}, trx)

					if err != nil {
						return err
					}

					if err := trx.Preload("Customer", &card).Error; err != nil {
						return err
					}

					err = queue.EnqueueTenantWebhook(
						ws.rc,
						ws.publisher,
						tenantId,
						models.WebhookDeliveryEventTypePaymentMethodAttached,
						card,
						trx,
					)

					if err != nil {
						log.Printf("Error occurred while enqueuing tenant webhook: %v", err)
					}
				}

				var subscription *models.Subscription
				var invoice *models.Invoice
				if subscriptionId != "" {
					subscription, err = ws.rc.SubscriptionRepository.Find(
						&models.Subscription{BaseModel: models.BaseModel{ID: subscriptionId}},
						&repositories.FindArgs{Trx: trx},
					)
					if err != nil {
						return err
					}
					if subscription == nil {
						return errors.New("subscription not found for the given subscription ID")
					}

					if invoiceId != "" {
						invoice, err = ws.rc.InvoiceRepository.Find(
							&models.Invoice{BaseModel: models.BaseModel{ID: invoiceId}},
							&repositories.FindArgs{Trx: trx},
						)
						if err != nil {
							return err
						}

						invoice.Status = models.InvoiceStatusPaid
						invoice.AmountPaid = planVersion.Amount
						invoice.AmountRemaining = 0
						invoice.PaidAt = utils.ToPtr(time.Now())

						_, err = ws.rc.InvoiceRepository.Update(invoice, trx)
						if err != nil {
							return err
						}

						startDate, endDate := utils.GetBillingPeriod(*subscription.CurrentBillingCycleEnd, subscription.Interval, subscription.IntervalCount)
						subscription.PaymentSourceID = &card.ID
						subscription.PaymentSourceType = utils.ToPtr(models.PaymentSourceTypeCard)
						subscription.CurrentBillingCycleStart = &startDate
						subscription.CurrentBillingCycleEnd = &endDate
						subscription.LatestInvoiceID = &invoice.ID
						subscription.InvoiceCount++
						_, err = ws.rc.SubscriptionRepository.Update(subscription, trx)
						if err != nil {
							return err
						}

						err = queue.EnqueueTenantWebhook(
							ws.rc,
							ws.publisher,
							tenantId,
							models.WebhookDeliveryEventTypeSubscriptionCreated,
							subscription,
							trx,
						)

						if err != nil {
							log.Printf("Error occurred while enqueuing tenant webhook: %v", err)
						}

						err = queue.EnqueueTenantWebhook(
							ws.rc,
							ws.publisher,
							tenantId,
							models.WebhookDeliveryEventTypeInvoicePaid,
							invoice,
							trx,
						)
						if err != nil {
							log.Printf("Error occurred while enqueuing tenant webhook: %v", err)
						}
					}
				} else {
					subscription = &models.Subscription{
						TenantID:          tenantId,
						CustomerID:        customer.ID,
						PlanID:            planVersion.PlanID,
						PlanVersionID:     planVersion.ID,
						PaymentSourceID:   &card.ID,
						PaymentSourceType: utils.ToPtr(models.PaymentSourceTypeCard),
						Interval:          planVersion.Interval,
						Amount:            planVersion.Amount,
						IntervalCount:     planVersion.IntervalCount,
						TrialPeriodDays:   planVersion.TrialPeriodDays,
						Currency:          planVersion.Currency,
						InvoiceLimit:      planVersion.InvoiceLimit,
						Status:            models.SubscriptionStatusActive,
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

					subscription.Code, err = utils.GenerateCode("SUB")
					if err != nil {
						return err
					}

					subscription, err = ws.rc.SubscriptionRepository.Create(subscription, trx)
					if err != nil {
						return err
					}
				}

				// for new subscriptions with no trial period, create an invoice and mark it as paid
				if (invoice == nil && subscriptionId != "") || (subscriptionId == "" && subscription.TrialPeriodDays == 0) {
					invoice = &models.Invoice{
						TenantID:        tenantId,
						SubscriptionID:  subscription.ID,
						CustomerID:      customer.ID,
						Status:          models.InvoiceStatusPaid,
						AmountDue:       planVersion.Amount,
						AmountPaid:      planVersion.Amount,
						AmountRemaining: 0,
						Currency:        planVersion.Currency,
						DueAt:           subscription.CurrentBillingCycleStart,
						PaidAt:          utils.ToPtr(time.Now()),
					}

					invoice.Code, err = utils.GenerateCode("INV")
					if err != nil {
						return err
					}

					invoice, err = ws.rc.InvoiceRepository.Create(invoice, trx)
					if err != nil {
						return err
					}

					subscription.LatestInvoiceID = &invoice.ID
					_, err = ws.rc.SubscriptionRepository.Update(subscription, trx)
					if err != nil {
						return err
					}

					err = queue.EnqueueTenantWebhook(
						ws.rc,
						ws.publisher,
						tenantId,
						models.WebhookDeliveryEventTypeInvoicePaid,
						invoice,
						trx,
					)
					if err != nil {
						log.Printf("Error occurred while enqueuing tenant webhook: %v", err)
					}
				}

				paymentIntent := &models.PaymentIntent{
					TenantID:                     tenantId,
					CustomerID:                   customer.ID,
					SubscriptionID:               subscription.ID,
					InvoiceID:                    subscription.LatestInvoiceID,
					PlanID:                       subscription.PlanID,
					PlanVersionID:                subscription.PlanVersionID,
					Reference:                    utils.OrStrings(orderReference, initiation.Reference),
					Amount:                       planVersion.Amount,
					Currency:                     planVersion.Currency,
					Status:                       models.PaymentIntentStatusSuccess,
					AttemptedAt:                  utils.ToPtr(time.Now()),
					PaymentSourceID:              &card.ID,
					PaymentSourceType:            &card.Type,
					ProviderTransactionID:        &orderId,
					ProviderTransactionReference: &orderReference,
				}

				paymentIntent.Code, err = utils.GenerateCode("PAY")
				_, err = ws.rc.PaymentIntentRepository.Create(paymentIntent, trx)
				if err != nil {
					return err
				}

				if subscriptionId == "" {
					enqueueSubscriptionEmail(ws.rc, ws.publisher, models.EmailTemplateSubscriptionCreated, subscription, string(models.EmailTemplateSubscriptionCreated)+":"+subscription.ID)
					if subscription.TrialPeriodDays > 0 {
						enqueueSubscriptionEmail(ws.rc, ws.publisher, models.EmailTemplateTrialStarted, subscription, string(models.EmailTemplateTrialStarted)+":"+subscription.ID)
					}
				}
				if subscription.TrialPeriodDays == 0 {
					enqueueSubscriptionEmail(ws.rc, ws.publisher, models.EmailTemplateSubscriptionActivated, subscription, string(models.EmailTemplateSubscriptionActivated)+":"+subscription.ID)
				}
				if invoice != nil {
					enqueueInvoiceEmail(ws.rc, ws.publisher, models.EmailTemplateInvoicePaid, invoice, string(models.EmailTemplateInvoicePaid)+":"+invoice.ID)
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
				}, trx)
				if err != nil {
					return err
				}

				initiation.Status = models.NombaInitiationStatusCompleted
				initiation.NombaTransactionId = &payload.Data.Transaction.TransactionID
				_, err = initiationRepository.Update(initiation, trx)
				if err != nil {
					return err
				}

				payload.Data.Subscription = subscription
				payload.Data.Transaction.IsSubscriptionPayment = true
				payload.Data.Order.IsSubscription = true
				payload.Data.Order.SubscriptionPlanCode = planVersion.Code
				payload.Data.Order.OrderReference = tenantOrderReference

				err = queue.EnqueueTenantWebhook(
					ws.rc,
					ws.publisher,
					tenantId,
					models.WebhookDeliveryEventTypeOrderSuccess,
					payload.Data,
					trx,
				)

				if err != nil {
					log.Printf("Error occurred while enqueuing tenant webhook: %v", err)
				}

			}
		}

		return nil
	})

	if err != nil {
		return err
	}
	return err
}

func (ws *WebhookService) handlePaymentFailed(payload nomba.NombaWebhookRequest) error {
	db := ws.rc.DB
	reason := utils.OrStrings(payload.Data.Transaction.ResponseCodeMessage, "unable to charge payment source")
	orderId := payload.Data.Order.OrderId
	orderReference := utils.OrStrings(payload.Data.Order.OrderReference, payload.Data.Transaction.MerchantTxRef)
	initiationRepository := ws.rc.NombaInitiationRepository

	err := db.Transaction(func(trx *gorm.DB) error {
		var tenantId string
		var invoice *models.Invoice
		var subscription *models.Subscription
		var paymentIntent *models.PaymentIntent
		var err error

		if orderReference != "" {
			initiation, err := initiationRepository.FindRaw(&repositories.FindArgs{
				Filter: repositories.NewQueryFilter().Where(
					"reference = ? OR nomba_order_id = ?", orderReference, orderId,
				),
				Trx: trx,
			})

			if err != nil {
				return err
			}

			if initiation == nil {
				return nil
			}

			if initiation.Status != models.NombaInitiationStatusPending {
				return nil
			}

			paymentIntent, err = ws.rc.PaymentIntentRepository.Find(
				&models.PaymentIntent{Reference: orderReference},
				&repositories.FindArgs{Trx: trx},
			)
			if err != nil {
				return err
			}
		}

		if paymentIntent != nil {
			tenantId = paymentIntent.TenantID
			paymentIntent.Status = models.PaymentIntentStatusFailed
			paymentIntent.FailureReason = &reason
			paymentIntent.FailedAt = utils.ToPtr(time.Now())
			if payload.Data.Transaction.TransactionID != "" {
				paymentIntent.ProviderTransactionID = &payload.Data.Transaction.TransactionID
			}
			if orderReference != "" {
				paymentIntent.ProviderTransactionReference = &orderReference
			}
			if _, err = ws.rc.PaymentIntentRepository.Update(paymentIntent, trx); err != nil {
				return err
			}

			if paymentIntent.InvoiceID != nil {
				invoice, err = ws.rc.InvoiceRepository.FindById(*paymentIntent.InvoiceID, &repositories.FindArgs{Trx: trx})
				if err != nil {
					return err
				}
			}

			subscription, err = ws.rc.SubscriptionRepository.FindById(paymentIntent.SubscriptionID, &repositories.FindArgs{Trx: trx})
			if err != nil {
				return err
			}
		} else {
			initiation, err := ws.rc.NombaInitiationRepository.FindRaw(&repositories.FindArgs{
				Filter: repositories.NewQueryFilter().Where(
					"reference = ? OR nomba_order_id = ?", orderReference, orderId,
				),
				Trx: trx,
			})
			if err != nil {
				return err
			}
			if initiation == nil || initiation.Purpose != models.NombaInitiationPurposeCardSubscriptionPayment {
				return nil
			}

			initiation.Status = models.NombaInitiationStatusFailed
			if _, err = ws.rc.NombaInitiationRepository.Update(initiation, trx); err != nil {
				return err
			}

			tenantId, _ = initiation.Metadata["nombaSubTenantId"].(string)
			invoiceId, _ := initiation.Metadata["nombaSubInvoiceId"].(string)
			subscriptionId, _ := initiation.Metadata["nombaSubSubscriptionId"].(string)

			if invoiceId != "" {
				invoice, err = ws.rc.InvoiceRepository.FindById(invoiceId, &repositories.FindArgs{Trx: trx})
				if err != nil {
					return err
				}
			}
			if subscriptionId != "" {
				subscription, err = ws.rc.SubscriptionRepository.FindById(subscriptionId, &repositories.FindArgs{Trx: trx})
				if err != nil {
					return err
				}
			}

			reference := utils.OrStrings(orderReference, initiation.Reference)
			if invoice != nil && subscription != nil && reference != "" {
				paymentIntent, err = ws.rc.PaymentIntentRepository.Find(
					&models.PaymentIntent{Reference: reference},
					&repositories.FindArgs{Trx: trx},
				)
				if err != nil {
					return err
				}
				if paymentIntent == nil {
					paymentIntent = &models.PaymentIntent{
						TenantID:       subscription.TenantID,
						CustomerID:     subscription.CustomerID,
						SubscriptionID: subscription.ID,
						InvoiceID:      &invoice.ID,
						PlanID:         subscription.PlanID,
						PlanVersionID:  subscription.PlanVersionID,
						Reference:      reference,
						Amount:         invoice.AmountDue,
						Currency:       invoice.Currency,
						Status:         models.PaymentIntentStatusFailed,
						FailureReason:  &reason,
						AttemptedAt:    utils.ToPtr(time.Now()),
						FailedAt:       utils.ToPtr(time.Now()),
					}
					paymentIntent.Code, err = utils.GenerateCode("PAY")
					if err != nil {
						return err
					}
					if payload.Data.Transaction.TransactionID != "" {
						paymentIntent.ProviderTransactionID = &payload.Data.Transaction.TransactionID
					}
					paymentIntent.ProviderTransactionReference = &reference
					if _, err = ws.rc.PaymentIntentRepository.Create(paymentIntent, trx); err != nil {
						return err
					}
				} else {
					paymentIntent.Status = models.PaymentIntentStatusFailed
					paymentIntent.FailureReason = &reason
					paymentIntent.FailedAt = utils.ToPtr(time.Now())
					if payload.Data.Transaction.TransactionID != "" {
						paymentIntent.ProviderTransactionID = &payload.Data.Transaction.TransactionID
					}
					paymentIntent.ProviderTransactionReference = &reference
					if _, err = ws.rc.PaymentIntentRepository.Update(paymentIntent, trx); err != nil {
						return err
					}
				}
			}
		}

		if invoice != nil {
			invoice.Status = models.InvoiceStatusFailed
			invoice.FailedAt = utils.ToPtr(time.Now())
			invoice.FailureReason = &reason
			if invoice.AttemptCount == 0 {
				invoice.AttemptCount = 1
			}
			if _, err = ws.rc.InvoiceRepository.Update(invoice, trx); err != nil {
				return err
			}
		}

		if subscription != nil {
			subscription.Status = models.SubscriptionStatusPaused
			subscription.PausedAt = utils.ToPtr(time.Now())
			if invoice != nil {
				subscription.LatestInvoiceID = &invoice.ID
			}
			if _, err = ws.rc.SubscriptionRepository.Update(subscription, trx); err != nil {
				return err
			}
			if tenantId == "" {
				tenantId = subscription.TenantID
			}
		}

		if tenantId != "" && invoice != nil {
			if err := queue.EnqueueTenantWebhook(ws.rc, ws.publisher, tenantId, models.WebhookDeliveryEventTypeInvoicePaymentFailed, invoice, trx); err != nil {
				log.Printf("Error occurred while enqueuing tenant webhook: %v", err)
			}
		}
		if tenantId != "" && subscription != nil {
			if err := queue.EnqueueTenantWebhook(ws.rc, ws.publisher, tenantId, models.WebhookDeliveryEventTypeSubscriptionPaused, subscription, trx); err != nil {
				log.Printf("Error occurred while enqueuing tenant webhook: %v", err)
			}
			enqueueSubscriptionPausedEmail(ws.rc, ws.publisher, subscription, invoice, reason)
		}

		return nil
	})

	if err != nil {
		return err
	}
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
