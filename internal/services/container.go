package services

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

type Container struct {
	AuthService                     *AuthService
	CustomerService                 *CustomerService
	PlanService                     *PlanService
	TransactionService              *TransactionService
	WebhookService                  *WebhookService
	SubscriptionService             *SubscriptionService
	InvoiceService                  *InvoiceService
	SubscriptionLifecycleService    *SubscriptionLifecycleService
	DirectDebitSubscriptionService  *DirectDebitSubscriptionService
}

func NewContainer(rc *repositories.Container, nombaProvider nomba.Provider, publisher *queue.Publisher) *Container {
	authService := NewAuthService(rc)
	customerService := NewCustomerService(rc)
	planService := NewPlanService(rc)
	transactionService := NewTransactionService(rc, nombaProvider, customerService)
	webhookService := NewWebhookService(rc, nombaProvider, publisher)
	subscriptionService := NewSubscriptionService(rc, planService, customerService, publisher, nombaProvider)
	invoiceService := NewInvoiceService(rc, nombaProvider, publisher)
	subscriptionLifecycleService := NewSubscriptionLifecycleService(rc, publisher)
	directDebitSubscriptionService := NewDirectDebitSubscriptionService(rc, nombaProvider, publisher)

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
	}
}
