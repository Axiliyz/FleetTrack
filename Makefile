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
	@docker run --rm \
		-v $(PWD):/app \
		-w /app \
		golangci/golangci-lint run ./...

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

migrate:
	@docker compose exec api migrate up 

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