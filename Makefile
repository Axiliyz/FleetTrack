include .env
export

# LOCAL

run:
	@go run ./cmd/api

build:
	@go build -o bin/$(APP_NAME) ./cmd/api

test:
	@go test ./...

fmt:
	@go fmt ./...

vet:
	@go vet ./...

lint:
	@golangci-lint run ./...

# DOCKER

docker-up:
	@docker compose up -d

docker-build:
	@docker compose build

docker-down:
	@docker compose down

docker-restart:
	@docker compose restart

docker-logs:
	@docker compose logs -f

# DB

db:
	@docker compose exec postgres psql -U $(DB_USER) -d $(DB_NAME)

migrate:
	@docker compose exec api migrate up 

# USEFUL

help:
	@echo "Commands:"
	@echo "make run"
	@echo "make build"
	@echo "make test"
	@echo "make up"
	@echo "make down"
	@echo "make lint"
	@echo "make fmt"

clean:
	@docker compose down -v
	@rm -rf bin/