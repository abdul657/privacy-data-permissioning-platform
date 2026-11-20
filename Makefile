.PHONY: run test test-integration vet fmt build up down
run:
	go run ./cmd/api
test:
	go test ./...
test-integration:
	go test ./internal/store -run TestPostgresConsentLifecycleAndTenantIsolation -v
vet:
	go vet ./...
fmt:
	gofmt -w ./cmd ./internal
build:
	go build ./cmd/api
up:
	docker compose up --build
down:
	docker compose down
