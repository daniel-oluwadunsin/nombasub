package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/cron"
	"github.com/daniel-oluwadunsin/nombasub/internal/db"
	"github.com/daniel-oluwadunsin/nombasub/internal/handlers"
	"github.com/daniel-oluwadunsin/nombasub/internal/mail"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/router"
	"github.com/daniel-oluwadunsin/nombasub/internal/services"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	if err := database.AutoMigrate(
		&models.Tenant{},
		&models.Customer{},
		&models.Plan{},
		&models.PlanVersion{},
		&models.Subscription{},
		&models.Invoice{},
		&models.PaymentSource{},
		&models.PaymentIntent{},
		&models.WebhookDelivery{},
		&models.WebhookDeliveryAttempt{},
		&models.NombaWebhookEvent{},
		&models.NombaInitiation{},
		&models.EmailDelivery{},
		&models.SettlementPayout{},
		&models.Settlement{},
		&models.Refund{},
	); err != nil {
		log.Fatalf("auto-migrate failed: %v", err)
	}

	nombaProvider, err := nomba.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize nomba provider: %v", err)
	}
	rc := repositories.NewContainer(database)

	mailer := mail.NewMailer(cfg.MailerUser, cfg.MailerPassword)

	mq, err := queue.NewConnection(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("rabbitmq connection failed: %v", err)
	}
	defer mq.Close()

	if err := mq.DeclareQueue(queue.SendTenantWebhookQueue); err != nil {
		log.Fatalf("failed to declare tenant webhook queue: %v", err)
	}
	if err := mq.DeclareQueue(queue.SendEmailQueue); err != nil {
		log.Fatalf("failed to declare email queue: %v", err)
	}
	publisher := queue.NewPublisher(mq)
	consumer := queue.NewConsumer(mq)
	consumer.Register(queue.SendTenantWebhookQueue, queue.SendTenantWebhookHandler(rc))
	consumer.Register(queue.SendEmailQueue, queue.SendEmailHandler(rc, mailer))
	consumer.Start()

	sc := services.NewContainer(rc, nombaProvider, publisher, cfg)
	handlers := handlers.New(sc)

	scheduler := cron.NewScheduler()
	if err := cron.RegisterSubscriptionLifecycleJobs(scheduler, sc.SubscriptionLifecycleService); err != nil {
		log.Fatalf("failed to register subscription lifecycle cron jobs: %v", err)
	}
	if err := cron.RegisterInvoiceProcessingJobs(scheduler, sc.InvoiceService); err != nil {
		log.Fatalf("failed to register invoice processing cron jobs: %v", err)
	}
	if err := cron.RegisterDirectDebitJobs(scheduler, sc.DirectDebitSubscriptionService); err != nil {
		log.Fatalf("failed to register direct debit cron jobs: %v", err)
	}
	if err := cron.RegisterSettlementJobs(scheduler, sc.SettlementService); err != nil {
		log.Fatalf("failed to register settlement cron jobs: %v", err)
	}
	scheduler.Start()
	defer scheduler.Stop()

	r := router.New(cfg, handlers, rc, sc)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
