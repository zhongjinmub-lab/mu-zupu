.PHONY: run tidy compose-up compose-down migrate-up migrate-down migrate-status release

run:
	go run ./cmd/server

tidy:
	go mod tidy

compose-up:
	docker compose -f deploy/docker-compose.yml up -d

compose-down:
	docker compose -f deploy/docker-compose.yml down

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

release:
	VERSION=$${VERSION:-dev} OUT_DIR=$${OUT_DIR:-dist} bash scripts/build_release.sh
