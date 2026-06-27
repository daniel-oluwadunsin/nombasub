package models

const (
	TableNameTenant                 = "tenants"
	TableNamePlan                   = "plans"
	TableNamePlanVersion            = "plan_versions"
	TableNameCustomer               = "customers"
	TableNameSubscription           = "subscriptions"
	TableNamePaymentSource          = "payment_sources"
	TableNameInvoices               = "invoices"
	TableNamePaymentIntent          = "payment_intents"
	TableNameWebhookDelivery        = "webhook_deliveries"        // for us to store the webhook deliveries sent from us to Tenants webhook endpoints
	TableNameWebhookDeliveryAttempt = "webhook_delivery_attempts" // for us to store the webhook delivery attempts sent from us to Tenants webhook endpoints
	TableNameNombaWebhookEvent      = "nomba_webhook_events"      // for us to store the webhook events sent by Nomba to our webhook endpoint
	TableAuditLog                   = "audit_logs"
	TableNombaInitiation            = "nomba_initiations"
)
