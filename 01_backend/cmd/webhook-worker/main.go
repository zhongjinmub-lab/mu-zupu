package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mu-agent-saas/internal/config"
	"mu-agent-saas/internal/module/webhook"
	"mu-agent-saas/pkg/database"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.NewPostgresPool(ctx, cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("connect database failed: %v", err)
	}
	defer db.Close()

	repo := webhook.NewRepository(db)
	service := webhook.NewService(repo, webhook.ServiceOptions{
		MaxRetries:       cfg.WebhookMaxRetries,
		RetryBaseSeconds: cfg.WebhookRetryBaseSeconds,
	})
	interval := time.Duration(cfg.WebhookWorkerIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 15 * time.Second
	}
	batchSize := cfg.WebhookWorkerBatchSize
	if batchSize <= 0 || batchSize > 100 {
		batchSize = 20
	}

	log.Printf("webhook worker started interval=%s batch_size=%d max_retries=%d", interval, batchSize, cfg.WebhookMaxRetries)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		processOnce(ctx, service, batchSize)
		select {
		case <-ctx.Done():
			log.Println("webhook worker stopped")
			return
		case <-ticker.C:
		}
	}
}

func processOnce(ctx context.Context, service webhook.Service, batchSize int) {
	out := service.RetryDue(ctx, batchSize)
	if out.Claimed == 0 && out.Failed == 0 {
		return
	}
	log.Printf("webhook retries claimed=%d processed=%d failed=%d", out.Claimed, out.Processed, out.Failed)
}
