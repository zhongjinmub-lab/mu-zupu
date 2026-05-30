package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mu-agent-saas/internal/config"
	filemodule "mu-agent-saas/internal/module/file"
	"mu-agent-saas/internal/module/kb"
	"mu-agent-saas/pkg/database"
	"mu-agent-saas/pkg/storage"
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

	storageClient, err := storage.NewMinIO(storage.Config{
		Endpoint:   cfg.StorageEndpoint,
		AccessKey:  cfg.StorageAccessKey,
		SecretKey:  cfg.StorageSecretKey,
		Bucket:     cfg.StorageBucket,
		UseSSL:     cfg.StorageUseSSL,
		PublicBase: cfg.StoragePublicBase,
	})
	if err != nil {
		log.Fatalf("init storage failed: %v", err)
	}

	kbRepo := kb.NewRepository(db)
	runner := kb.NewDocumentJobRunner(kbRepo, filemodule.NewRepository(db), storageClient)
	interval := time.Duration(cfg.DocumentWorkerIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 10 * time.Second
	}
	batchSize := cfg.DocumentWorkerBatchSize
	if batchSize <= 0 || batchSize > 50 {
		batchSize = 5
	}

	log.Printf("document worker started interval=%s batch_size=%d", interval, batchSize)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		processOnce(ctx, kbRepo, runner, batchSize)
		select {
		case <-ctx.Done():
			log.Println("document worker stopped")
			return
		case <-ticker.C:
		}
	}
}

func processOnce(ctx context.Context, repo kb.Repository, runner kb.DocumentJobRunner, batchSize int) {
	jobs, err := repo.ClaimAnyDocumentJobs(ctx, batchSize)
	if err != nil {
		log.Printf("claim document jobs failed: %v", err)
		return
	}
	if len(jobs) == 0 {
		return
	}
	out := runner.Run(ctx, jobs)
	log.Printf("document jobs processed=%d failed=%d claimed=%d", out.Processed, out.Failed, len(jobs))
}
