package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

func enqueueSubscriptionEmail(rc *repositories.Container, publisher *queue.Publisher, templateName models.EmailTemplateName, subscription *models.Subscription, idempotencyKey string) {
	ctx, err := subscriptionEmailContext(rc, subscription)
	if err != nil {
		log.Printf("subscription email context failed for subscription %s: %v", subscription.ID, err)
		return
	}
	applyTemplateCopy(templateName, &ctx)

	if err := queue.EnqueueEmail(rc, publisher, ctx.CustomerEmail, emailSubject(templateName, ctx), templateName, ctx, idempotencyKey); err != nil {
		log.Printf("subscription email enqueue failed for subscription %s template %s: %v", subscription.ID, templateName, err)
	}
}

func enqueueInvoiceEmail(rc *repositories.Container, publisher *queue.Publisher, templateName models.EmailTemplateName, invoice *models.Invoice, idempotencyKey string) {
	ctx, err := invoiceEmailContext(rc, invoice)
	if err != nil {
		log.Printf("invoice email context failed for invoice %s: %v", invoice.ID, err)
		return
	}
	applyTemplateCopy(templateName, &ctx)

	if err := queue.EnqueueEmail(rc, publisher, ctx.CustomerEmail, emailSubject(templateName, ctx), templateName, ctx, idempotencyKey); err != nil {
		log.Printf("invoice email enqueue failed for invoice %s template %s: %v", invoice.ID, templateName, err)
	}
}

func enqueueCheckoutEmail(rc *repositories.Container, publisher *queue.Publisher, invoice *models.Invoice, checkoutURL string) {
	ctx, err := invoiceEmailContext(rc, invoice)
	if err != nil {
		log.Printf("checkout email context failed for invoice %s: %v", invoice.ID, err)
		return
	}
	ctx.CheckoutURL = checkoutURL
	ctx.PrimaryActionURL = checkoutURL
	applyTemplateCopy(models.EmailTemplateCheckoutPaymentRequired, &ctx)

	if err := queue.EnqueueEmail(
		rc,
		publisher,
		ctx.CustomerEmail,
		emailSubject(models.EmailTemplateCheckoutPaymentRequired, ctx),
		models.EmailTemplateCheckoutPaymentRequired,
		ctx,
		fmt.Sprintf("%s:%s", models.EmailTemplateCheckoutPaymentRequired, invoice.ID),
	); err != nil {
		log.Printf("checkout email enqueue failed for invoice %s: %v", invoice.ID, err)
	}
}

func enqueueCardExpiringEmail(rc *repositories.Container, publisher *queue.Publisher, subscription *models.Subscription, paymentSource *models.PaymentSource) {
	ctx, err := subscriptionEmailContext(rc, subscription)
	if err != nil {
		log.Printf("card expiring email context failed for subscription %s: %v", subscription.ID, err)
		return
	}
	if paymentSource.Card != nil {
		if paymentSource.Card.Last4Digits != nil {
			ctx.CardLast4 = *paymentSource.Card.Last4Digits
		}
		if paymentSource.Card.ExpiryMonth != nil && paymentSource.Card.ExpiryYear != nil {
			ctx.CardExpiry = fmt.Sprintf("%s/%s", *paymentSource.Card.ExpiryMonth, *paymentSource.Card.ExpiryYear)
		}
	}
	applyTemplateCopy(models.EmailTemplateSubscriptionCardExpiring, &ctx)

	if err := queue.EnqueueEmail(
		rc,
		publisher,
		ctx.CustomerEmail,
		emailSubject(models.EmailTemplateSubscriptionCardExpiring, ctx),
		models.EmailTemplateSubscriptionCardExpiring,
		ctx,
		fmt.Sprintf("%s:%s:%s", models.EmailTemplateSubscriptionCardExpiring, subscription.ID, paymentSource.ID),
	); err != nil {
		log.Printf("card expiring email enqueue failed for subscription %s: %v", subscription.ID, err)
	}
}

func enqueueSubscriptionPausedEmail(rc *repositories.Container, publisher *queue.Publisher, subscription *models.Subscription, invoice *models.Invoice, reason string) {
	var ctx models.EmailContext
	var err error
	if invoice != nil {
		ctx, err = invoiceEmailContext(rc, invoice)
	} else {
		ctx, err = subscriptionEmailContext(rc, subscription)
	}
	if err != nil {
		log.Printf("subscription paused email context failed for subscription %s: %v", subscription.ID, err)
		return
	}

	applyTemplateCopy(models.EmailTemplateSubscriptionPaused, &ctx)
	if reason != "" {
		ctx.Body = fmt.Sprintf("%s Reason: %s.", ctx.Body, reason)
		ctx.SecondaryNote = fmt.Sprintf("Failure reason: %s", reason)
	}

	entityID := subscription.ID
	if invoice != nil {
		entityID = invoice.ID
	}

	if err := queue.EnqueueEmail(
		rc,
		publisher,
		ctx.CustomerEmail,
		emailSubject(models.EmailTemplateSubscriptionPaused, ctx),
		models.EmailTemplateSubscriptionPaused,
		ctx,
		fmt.Sprintf("%s:%s", models.EmailTemplateSubscriptionPaused, entityID),
	); err != nil {
		log.Printf("subscription paused email enqueue failed for subscription %s: %v", subscription.ID, err)
	}
}

func subscriptionEmailContext(rc *repositories.Container, subscription *models.Subscription) (models.EmailContext, error) {
	customer, err := rc.CustomerRepository.FindById(subscription.CustomerID, nil)
	if err != nil {
		return models.EmailContext{}, err
	}
	plan, err := rc.PlanRepository.FindById(subscription.PlanID, nil)
	if err != nil {
		return models.EmailContext{}, err
	}
	tenant, err := rc.TenantRepository.FindById(subscription.TenantID, nil)
	if err != nil {
		return models.EmailContext{}, err
	}
	if customer == nil || plan == nil || tenant == nil {
		return models.EmailContext{}, fmt.Errorf("missing customer, plan, or tenant")
	}

	ctx := models.EmailContext{
		GreetingName:       customerDisplayName(customer),
		BusinessName:       valueOrDefault(tenant.BusinessName, "Nomba merchant"),
		CustomerEmail:      customer.Email,
		PlanName:           plan.Name,
		PlanCode:           plan.Code,
		SubscriptionCode:   subscription.Code,
		SubscriptionStatus: string(subscription.Status),
		Amount:             formatAmount(subscription.Amount, subscription.Currency),
		Currency:           subscription.Currency,
		TrialStartDate:     formatDate(subscription.TrialStartDate),
		TrialEndDate:       formatDate(subscription.TrialEndDate),
		DueDate:            formatDate(subscription.CurrentBillingCycleStart),
		BillingPeriod:      formatDateRange(subscription.CurrentBillingCycleStart, subscription.CurrentBillingCycleEnd),
	}

	return ctx, nil
}

func invoiceEmailContext(rc *repositories.Container, invoice *models.Invoice) (models.EmailContext, error) {
	subscription, err := rc.SubscriptionRepository.FindById(invoice.SubscriptionID, nil)
	if err != nil {
		return models.EmailContext{}, err
	}
	if subscription == nil {
		return models.EmailContext{}, fmt.Errorf("subscription not found")
	}

	ctx, err := subscriptionEmailContext(rc, subscription)
	if err != nil {
		return models.EmailContext{}, err
	}

	ctx.InvoiceCode = invoice.Code
	ctx.InvoiceStatus = string(invoice.Status)
	ctx.Amount = formatAmount(invoice.AmountDue, invoice.Currency)
	ctx.Currency = invoice.Currency
	ctx.DueDate = formatDate(invoice.DueAt)
	ctx.BillingPeriod = formatDateRange(invoice.BillingPeriodStart, invoice.BillingPeriodEnd)
	ctx.PaymentDate = formatDate(invoice.PaidAt)
	ctx.ReceiptReference = invoice.Code
	ctx.CheckoutURL = valueOrDefault(invoice.CheckoutLink, "")

	return ctx, nil
}

func applyTemplateCopy(templateName models.EmailTemplateName, ctx *models.EmailContext) {
	switch templateName {
	case models.EmailTemplateSubscriptionCreated:
		ctx.Title = "Your subscription has started"
		ctx.Preheader = "Your subscription is set up and ready for the next billing step."
		ctx.Intro = "Your subscription has been created successfully."
		ctx.Body = "We have set up your subscription. If payment or mandate approval is needed, you will receive the next step shortly."
		ctx.SecondaryNote = "If your plan includes a trial, billing starts after the trial period ends."
	case models.EmailTemplateCheckoutPaymentRequired:
		ctx.Title = "Complete your first subscription payment"
		ctx.Preheader = "Use your secure Nomba checkout link to activate billing."
		ctx.Intro = "Your subscription is ready. Complete the first payment to continue."
		ctx.Body = "Use the secure checkout link below to pay your first invoice and save your card for future subscription payments."
		ctx.PrimaryActionLabel = "Pay with Nomba Checkout"
	case models.EmailTemplateSubscriptionActivated:
		ctx.Title = "Your subscription is active"
		ctx.Preheader = "Your first payment was received and your subscription is active."
		ctx.Intro = "Your subscription is now active."
		ctx.Body = "Your first payment was successful and your subscription is ready to continue."
	case models.EmailTemplateTrialStarted:
		ctx.Title = "Your trial has started"
		ctx.Preheader = "You are now on trial for this subscription."
		ctx.Intro = "Your trial period has started."
		ctx.Body = "You can use the subscription during the trial period. Billing starts when the trial ends."
	case models.EmailTemplateTrialEndingSoon:
		ctx.Title = "Your trial is ending soon"
		ctx.Preheader = "Your subscription trial will end soon."
		ctx.Intro = "Your trial is almost over."
		ctx.Body = "Your saved payment method will be charged when paid billing begins."
	case models.EmailTemplateTrialEndedBillingStarted:
		ctx.Title = "Paid billing has started"
		ctx.Preheader = "Your trial has ended and billing has started."
		ctx.Intro = "Your trial has ended."
		ctx.Body = "Paid billing has now started for your subscription. Your first invoice is due now."
	case models.EmailTemplateUpcomingInvoice:
		ctx.Title = "Upcoming subscription invoice"
		ctx.Preheader = "You will be charged soon for your subscription."
		ctx.Intro = "Your next invoice is coming up."
		ctx.Body = "We will charge your saved payment method on the due date."
	case models.EmailTemplateInvoiceCreated:
		ctx.Title = "Your invoice is ready"
		ctx.Preheader = "A subscription invoice has been created."
		ctx.Intro = "Your invoice has been opened."
		ctx.Body = "The invoice details are below. We will attempt payment using your saved payment method."
	case models.EmailTemplatePaymentSuccessful:
		ctx.Title = "Payment successful"
		ctx.Preheader = "Your subscription payment was received."
		ctx.Intro = "Your payment was successful."
		ctx.Body = "We have received your subscription payment."
	case models.EmailTemplatePaymentReceipt:
		ctx.Title = "Your payment receipt"
		ctx.Preheader = "Receipt details for your subscription payment."
		ctx.Intro = "Here is your payment receipt."
		ctx.Body = "Keep this receipt for your records."
	case models.EmailTemplateInvoicePaid:
		ctx.Title = "Invoice settled"
		ctx.Preheader = "Your subscription invoice has been paid."
		ctx.Intro = "Your invoice is fully paid."
		ctx.Body = "No further action is needed for this invoice."
	case models.EmailTemplateSubscriptionCardExpiring:
		ctx.Title = "Your subscription card is expiring soon"
		ctx.Preheader = "Update your payment method to avoid failed subscription payments."
		ctx.Intro = "Your saved card is expiring soon."
		ctx.Body = "Please update your subscription payment method before the card expires."
	case models.EmailTemplateSubscriptionPaused:
		ctx.Title = "Your subscription is paused"
		ctx.Preheader = "We could not complete your subscription payment."
		ctx.Intro = "Your subscription has been paused."
		ctx.Body = "We could not complete the latest invoice payment, so billing has been paused until the payment issue is resolved."
	case models.EmailTemplateSubscriptionCanceled:
		ctx.Title = "Your subscription has been canceled"
		ctx.Preheader = "Your subscription billing has been stopped."
		ctx.Intro = "Your subscription has been canceled."
		ctx.Body = "Billing for this subscription has stopped. Any open invoice for this subscription has been closed."
	}
}

func emailSubject(templateName models.EmailTemplateName, ctx models.EmailContext) string {
	switch templateName {
	case models.EmailTemplateCheckoutPaymentRequired:
		return fmt.Sprintf("Complete payment for %s", ctx.PlanName)
	case models.EmailTemplateUpcomingInvoice:
		return fmt.Sprintf("Upcoming invoice for %s", ctx.PlanName)
	case models.EmailTemplateInvoiceCreated:
		return fmt.Sprintf("Invoice %s is ready", ctx.InvoiceCode)
	case models.EmailTemplatePaymentReceipt:
		return fmt.Sprintf("Receipt for invoice %s", ctx.InvoiceCode)
	case models.EmailTemplateSubscriptionCardExpiring:
		return "Your subscription card is expiring soon"
	case models.EmailTemplateSubscriptionPaused:
		return "Your subscription has been paused"
	case models.EmailTemplateSubscriptionCanceled:
		return "Your subscription has been canceled"
	default:
		return ctx.Title
	}
}

func customerDisplayName(customer *models.Customer) string {
	if customer.Name != nil && strings.TrimSpace(*customer.Name) != "" {
		return *customer.Name
	}
	if customer.Email == "" {
		return "there"
	}
	return strings.Split(customer.Email, "@")[0]
}

func valueOrDefault(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func formatAmount(amount int64, currency string) string {
	return fmt.Sprintf("%s %.2f", currency, float64(amount)/100)
}

func formatDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("Jan 2, 2006")
}

func formatDateRange(start, end *time.Time) string {
	if start == nil || end == nil {
		return ""
	}
	return fmt.Sprintf("%s - %s", formatDate(start), formatDate(end))
}
