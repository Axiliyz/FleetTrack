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

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go -o docs

check: fmt vet test lint


# DOCKER

docker-build:
	@docker compose build
	@docker compose up -d

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
		echo "отсутствует параметр seq, пример: make migrate-create seq=init"; \
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
	@if [ -z "$(action)" ]; then \
		echo "отсутствует параметр action, пример: make migrate-action action=\"up 1\""; \
		exit 1; \
	fi
	@docker compose run --rm fleettrack-postgres-migrate \
		-path /migrations \
		-database postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}?sslmode=disable \
		$(action)

# LOADTEST

LT_COMPOSE = docker compose -f docker-compose.yml -f docker-compose.loadtest.yml
LT_DB = fleettrack_loadtest
BASELINE ?= $(HOME)/fleettrack-loadtest/baseline.dump
SCRIPT ?= telemetry_ingest.js

loadtest-snapshot:
	@mkdir -p $(dir $(BASELINE))
	@docker compose exec -T postgres pg_dump -U $(DB_USER) -d $(DB_NAME) -Fc > $(BASELINE)
	@echo "Снимок сохранён: $(BASELINE)"

loadtest-reset:
	@$(LT_COMPOSE) stop api
	@docker compose exec -T postgres dropdb -U $(DB_USER) --if-exists $(LT_DB)
	@docker compose exec -T postgres createdb -U $(DB_USER) $(LT_DB)
	@docker compose exec -T postgres pg_restore -U $(DB_USER) -d $(LT_DB) < $(BASELINE)
	@$(LT_COMPOSE) up -d api
	@echo "Тестовая база восстановлена из $(BASELINE)"

loadtest-run:
	@$(MAKE) loadtest-reset
	@$(LT_COMPOSE) --profile loadtest run --rm k6 run /loadtest/$(SCRIPT)

loadtest-stop:
	@docker compose up -d api
	@echo "API снова подключён к рабочей базе $(DB_NAME)"

loadtest-check:
	@docker compose exec -T postgres psql -U $(DB_USER) -d $(LT_DB) -At -c "select count(*) as telemetry_rows from telemetry;"

loadtest-pairs:
	@docker compose exec -T postgres psql -U $(DB_USER) -d $(LT_DB) -At -c "select json_agg(json_build_object('device_id', device_id, 'vehicle_id', vehicle_id)) from device_assignments;" > loadtest/pairs.json
	@echo "Пары записаны в loadtest/pairs.json"

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
	@echo "make swagger"
	@echo "make check"
	@echo "make loadtest-snapshot   (снимок dev-базы в BASELINE)"
	@echo "make loadtest-reset      (восстановить тестовую базу из снимка)"
	@echo "make loadtest-run SCRIPT=telemetry_ingest.js   (сброс + k6)"
	@echo "make loadtest-stop       (вернуть API на рабочую базу после тестов)"
	@echo "make loadtest-check      (счётчик строк телеметрии в тестовой базе)"
	@echo "make loadtest-pairs      (выгрузить пары device/vehicle для k6)"
	@echo "make clean"

clean:
	@read -p "Очистить окружение? [y/n]: " ans; \
	if [ "$$ans" = "y" ]; then \
		docker compose down -v; \
		rm -rf bin/; \
		echo "Окружение очищено"; \
	else \
		echo "Очистка отменена"; \
	fi