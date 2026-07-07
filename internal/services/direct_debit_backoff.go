package services

import (
	"log"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

// Per-mandate exponential backoff for the direct-debit mandate poll.
//
// Without this, the cron would hit Nomba once per pending mandate on every tick.
// That scales badly: a mandate that never gets approved by the customer would be
// polled forever, and every failed round-trip counts against our Nomba rate limit.
//
// The scheduler behavior with backoff in place:
//   1–10 attempts  -> re-poll after 1  min
//   11–30          -> re-poll after 5  min
//   31–60          -> re-poll after 30 min
//   60+            -> re-poll after 60 min
//   after 30 days  -> mark the initiation failed and stop polling entirely.
//
// The outer cron tick can safely fire at any interval (e.g. every minute) — this
// function is the one that decides which mandates are actually due.

const maxMandatePollAge = 30 * 24 * time.Hour

func nextPollDelay(attempts int) time.Duration {
	switch {
	case attempts < 10:
		return 1 * time.Minute
	case attempts < 30:
		return 5 * time.Minute
	case attempts < 60:
		return 30 * time.Minute
	default:
		return 60 * time.Minute
	}
}

func mandateReadyToPoll(initiation *models.NombaInitiation, now time.Time) bool {
	if initiation.LastPolledAt == nil {
		return true
	}
	return now.After(initiation.LastPolledAt.Add(nextPollDelay(initiation.PollAttempts)))
}

// PollPendingMandates loads every pending direct-debit initiation, expires ones
// older than maxMandatePollAge, and calls processPendingMandate for those whose
// backoff window has elapsed.
func (s *DirectDebitSubscriptionService) PollPendingMandates() {
	initiations, err := s.rc.NombaInitiationRepository.FindManyRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"purpose = ? AND status = ?",
			models.NombaInitiationPurposeDirectDebitSubscription,
			models.NombaInitiationStatusPending,
		),
	})
	if err != nil {
		log.Printf("direct debit poll cron: failed to load pending initiations: %v", err)
		return
	}

	now := time.Now()
	for _, initiation := range initiations {
		initiation := initiation

		if now.Sub(initiation.CreatedAt) > maxMandatePollAge {
			initiation.Status = models.NombaInitiationStatusFailed
			if _, err := s.rc.NombaInitiationRepository.Update(&initiation, nil); err != nil {
				log.Printf("direct debit poll: failed to expire mandate=%s: %v", initiation.Reference, err)
			} else {
				log.Printf("direct debit poll: expired mandate=%s after %s pending", initiation.Reference, now.Sub(initiation.CreatedAt))
			}
			continue
		}

		if !mandateReadyToPoll(&initiation, now) {
			continue
		}

		initiation.LastPolledAt = &now
		initiation.PollAttempts++
		if _, err := s.rc.NombaInitiationRepository.Update(&initiation, nil); err != nil {
			log.Printf("direct debit poll: failed to update poll bookkeeping for mandate=%s: %v", initiation.Reference, err)
			continue
		}

		if err := s.processPendingMandate(&initiation); err != nil {
			log.Printf("direct debit poll: processPendingMandate failed for mandate=%s (attempt %d): %v",
				initiation.Reference, initiation.PollAttempts, err)
		}
	}
}
