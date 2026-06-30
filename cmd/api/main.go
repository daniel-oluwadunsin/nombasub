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
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
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
	); err != nil {
		log.Fatalf("auto-migrate failed: %v", err)
	}

	rc := repositories.NewContainer(database)
	sc := services.NewContainer(rc)
	handlers := handlers.New(sc)

	mq, err := queue.NewConnection(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("rabbitmq connection failed: %v", err)
	}
	defer mq.Close()

	publisher := queue.NewPublisher(mq)
	consumer := queue.NewConsumer(mq)
	_ = publisher
	_ = consumer

	scheduler := cron.NewScheduler()
	if err := scheduler.Register("0 * * * * *", "example", cron.ExampleJob); err != nil {
		log.Fatalf("failed to register cron job: %v", err)
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
