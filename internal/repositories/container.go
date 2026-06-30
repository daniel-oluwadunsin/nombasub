package repositories

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"gorm.io/gorm"
)

type Container struct {
	DB                               *gorm.DB
	TenantRepository                 *Repository[models.Tenant]
	CustomerRepository               *Repository[models.Customer]
	PlanRepository                   *Repository[models.Plan]
	PlanVersionRepository            *Repository[models.PlanVersion]
	SubscriptionRepository           *Repository[models.Subscription]
	InvoiceRepository                *Repository[models.Invoice]
	PaymentSourceRepository          *Repository[models.PaymentSource]
	PaymentIntentRepository          *Repository[models.PaymentIntent]
	WebhookDeliveryRepository        *Repository[models.WebhookDelivery]
	WebhookDeliveryAttemptRepository *Repository[models.WebhookDeliveryAttempt]
	NombaWebhookEventRepository      *Repository[models.NombaWebhookEvent]
	NombaInitiationRepository        *Repository[models.NombaInitiation]
	SettlementRepository             *Repository[models.Settlement]
}

func NewContainer(db *gorm.DB) *Container {
	return &Container{
		DB:                               db,
		TenantRepository:                 New[models.Tenant](db, models.TableNameTenant),
		CustomerRepository:               New[models.Customer](db, models.TableNameCustomer),
		PlanRepository:                   New[models.Plan](db, models.TableNamePlan),
		PlanVersionRepository:            New[models.PlanVersion](db, models.TableNamePlanVersion),
		SubscriptionRepository:           New[models.Subscription](db, models.TableNameSubscription),
		InvoiceRepository:                New[models.Invoice](db, models.TableNameInvoices),
		PaymentSourceRepository:          New[models.PaymentSource](db, models.TableNamePaymentSource),
		PaymentIntentRepository:          New[models.PaymentIntent](db, models.TableNamePaymentIntent),
		WebhookDeliveryRepository:        New[models.WebhookDelivery](db, models.TableNameWebhookDelivery),
		WebhookDeliveryAttemptRepository: New[models.WebhookDeliveryAttempt](db, models.TableNameWebhookDeliveryAttempt),
		NombaWebhookEventRepository:      New[models.NombaWebhookEvent](db, models.TableNameNombaWebhookEvent),
		NombaInitiationRepository:        New[models.NombaInitiation](db, models.TableNameNombaInitiation),
		SettlementRepository:             New[models.Settlement](db, models.TableNameSettlement),
	}
}
