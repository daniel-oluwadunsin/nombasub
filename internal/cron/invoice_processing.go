package cron

import "github.com/daniel-oluwadunsin/nombasub/internal/services"

func RegisterInvoiceProcessingJobs(scheduler *Scheduler, invoiceService *services.InvoiceService) error {
	if err := scheduler.Register(CronExpressionEveryThreeHours, "invoice-upcoming", invoiceService.CreateUpcomingInvoices); err != nil {
		return err
	}

	return scheduler.Register(CronExpressionEveryThreeHours, "invoice-processing", invoiceService.ProcessDueInvoices)
}

func RegisterSubscriptionLifecycleJobs(scheduler *Scheduler, subscriptionLifecycleService *services.SubscriptionLifecycleService) error {
	if err := scheduler.Register(CronExpressionEveryThreeHours, "subscription-trials", subscriptionLifecycleService.ProcessTrials); err != nil {
		return err
	}

	return scheduler.Register(CronExpressionEveryThreeHours, "subscription-card-expirations", subscriptionLifecycleService.ProcessCardExpirations)
}
