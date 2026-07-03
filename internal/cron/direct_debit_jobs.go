package cron

import "github.com/daniel-oluwadunsin/nombasub/internal/services"

func RegisterDirectDebitJobs(scheduler *Scheduler, svc *services.DirectDebitSubscriptionService) error {
	return scheduler.Register(CronExpressionEveryThirtyMins, "direct-debit-mandate-poll", svc.PollPendingMandates)
}
