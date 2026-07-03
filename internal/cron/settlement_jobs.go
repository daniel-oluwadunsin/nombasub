package cron

import "github.com/daniel-oluwadunsin/nombasub/internal/services"

func RegisterSettlementJobs(scheduler *Scheduler, svc *services.SettlementService) error {
	return scheduler.Register(CronExpressionWeekdayMorning, "settlement-payout", svc.ProcessDueSettlements)
}
