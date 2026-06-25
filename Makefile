include .env
export

export PROJECT_ROOT=${shell pwd}

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

docker-build:
	@docker compose build

docker-up:
	@docker compose up -d

docker-down:
	@docker compose down

docker-restart:
	@docker compose restart

docker-logs:
	@docker compose logs -f

# DB

db:
	@docker compose exec postgres psql -U $(DB_USER) -d $(DB_NAME)

migrate-create:
	@if [ -z "$(seq)" ]; then \
		echo "отсутствует параметр seq, ex: make migrate-create seq=init"; \
		exit 1; \
	fi
	@docker compose run --rm fleettrack-postgres-migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq $(seq)

migrate-up:
	@$(MAKE) migrate-action action=up
migrate-down:
	@$(MAKE) migrate-action action=down

migrate-action:
	@docker compose run --rm fleettrack-postgres-migrate \
		-path /migrations \
		-database postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}?sslmode=disable \
		$(action)

# USEFUL

help:
	@echo "Commands:"
	@echo "make run"
	@echo "make build"
	@echo "make test"
	@echo "make docker-up"
	@echo "make docker-down"
	@echo "make lint"
	@echo "make fmt"

clean:
	@read -p "Очистить окружение? [y/n]: " ans; \
	if [ "$$ans" = "y" ]; then \
		docker compose down -v; \
		rm -rf bin/; \
		echo "Окружение очищено"; \
	else \
		echo "Очистка отменена"; \
	fi