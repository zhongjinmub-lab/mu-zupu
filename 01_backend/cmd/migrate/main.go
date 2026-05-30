package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"mu-agent-saas/internal/config"
	"mu-agent-saas/internal/migrate"
	"mu-agent-saas/pkg/database"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./cmd/migrate [up|down|status]")
	}
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "migrations"
	}
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := database.NewPostgresPool(ctx, cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("connect database failed: %v", err)
	}
	defer db.Close()

	switch os.Args[1] {
	case "up":
		items, err := migrate.Up(ctx, db, dir)
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range items {
			fmt.Printf("applied %s_%s\n", item.Version, item.Name)
		}
		if len(items) == 0 {
			fmt.Println("no pending migrations")
		}
	case "down":
		item, err := migrate.Down(ctx, db, dir)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("rolled back %s_%s\n", item.Version, item.Name)
	case "status":
		items, err := migrate.Status(ctx, db, dir)
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range items {
			state := "pending"
			if item.AppliedAt != nil {
				state = "applied"
			}
			fmt.Printf("%s_%s %s\n", item.Version, item.Name, state)
		}
	default:
		log.Fatalf("unknown command %q", os.Args[1])
	}
}
