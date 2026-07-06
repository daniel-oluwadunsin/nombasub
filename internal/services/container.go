package services

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

type Container struct {
	AuthService                    *AuthService
	CustomerService                *CustomerService
	PlanService                    *PlanService
	TransactionService             *TransactionService
	WebhookService                 *WebhookService
	SubscriptionService            *SubscriptionService
	InvoiceService                 *InvoiceService
	SubscriptionLifecycleService   *SubscriptionLifecycleService
	DirectDebitSubscriptionService *DirectDebitSubscriptionService
	SettlementService              *SettlementService
	DashboardAnalyticsService      *DashboardAnalyticsService
	WebhookDeliveryService         *WebhookDeliveryService
}

func NewContainer(rc *repositories.Container, nombaProvider nomba.Provider, publisher *queue.Publisher, cfg *config.Config) *Container {
	authService := NewAuthService(rc, cfg)
	customerService := NewCustomerService(rc, publisher)
	planService := NewPlanService(rc)
	transactionService := NewTransactionService(rc, nombaProvider, customerService, publisher)
	webhookService := NewWebhookService(rc, nombaProvider, publisher)
	invoiceService := NewInvoiceService(rc, nombaProvider, publisher)
	subscriptionService := NewSubscriptionService(rc, planService, customerService, invoiceService, publisher, nombaProvider)
	subscriptionLifecycleService := NewSubscriptionLifecycleService(rc, publisher)
	directDebitSubscriptionService := NewDirectDebitSubscriptionService(rc, nombaProvider, publisher)
	settlementService := NewSettlementService(rc, nombaProvider, publisher, cfg)
	dashboardAnalyticsService := NewDashboardAnalyticsService(rc)
	webhookDeliveryService := NewWebhookDeliveryService(rc, publisher)

	return &Container{
		authService,
		customerService,
		planService,
		transactionService,
		webhookService,
		subscriptionService,
		invoiceService,
		subscriptionLifecycleService,
		directDebitSubscriptionService,
		settlementService,
		dashboardAnalyticsService,
		webhookDeliveryService,
	}
}
