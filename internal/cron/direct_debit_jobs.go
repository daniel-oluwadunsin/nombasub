package cron

import "github.com/daniel-oluwadunsin/nombasub/internal/services"

func RegisterDirectDebitJobs(scheduler *Scheduler, svc *services.DirectDebitSubscriptionService) error {
	// TEMP: every 1 minute for testing — revert to CronExpressionEveryThirtyMins before committing.
	return scheduler.Register("0 * * * * *", "direct-debit-mandate-poll", svc.PollPendingMandates)
}
