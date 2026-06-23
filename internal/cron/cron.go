package cron

import (
	"log"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	c *cron.Cron
}

func NewScheduler() *Scheduler {
	return &Scheduler{c: cron.New(cron.WithSeconds())}
}

// Register adds a job. spec is a 6-field cron expression (with seconds).
func (s *Scheduler) Register(spec, name string, fn func()) error {
	_, err := s.c.AddFunc(spec, func() {
		log.Printf("cron: running %s", name)
		fn()
	})
	return err
}

func (s *Scheduler) Start() {
	s.c.Start()
}

func (s *Scheduler) Stop() {
	s.c.Stop()
}
