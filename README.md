# FleetTrack

Fleet management system backend. Collects vehicle telemetry data, tracks device assignments, and manages organization structures.

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.26.3 (for local development)

### Running with Docker

```bash
# Start all services (PostgreSQL, API, migrations)
docker compose up --build

# API will be available at http://localhost:8080
```

### Local Development

```bash
# Start only PostgreSQL
docker compose up postgres -d

# Install dependencies
go mod tidy

# Run API
make run
```

## Project Structure

```
cmd/api/              # Application entry point
internal/
  ├── handler/        # HTTP handlers & error mapping
  ├── service/        # Business logic & validation
  ├── repository/     # Data persistence layer
  ├── model/          # Domain models & errors
  ├── logger/         # Logging interface
  └── middleware/     # HTTP middleware
migrations/           # Database migrations (SQL)
docker-compose.yml    # Service orchestration
```

## API Endpoints

### POST /telemetry
Save vehicle telemetry data.

**Request:**
```json
{
  "vehicle_id": 1,
  "device_id": 1,
  "lat": 55.7558,
  "lon": 37.6173,
  "fuel": 45.5
}
```

**Response:**
```json
{
  "status": "success",
  "message": "Telemetry saved",
  "request_id": "uuid",
  "data": {
    "telemetry_id": 123,
    "vehicle_id": 1,
    "device_id": 1,
    "received_at": "2026-06-29T10:30:00Z"
  }
}
```

## Database

### Setup Test Data

```bash
# Connect to PostgreSQL
make db

# Or manually via Docker
docker exec -it fleettrack-postgres-1 psql -U postgres -d fleettrack
```

Insert test data:
```sql
INSERT INTO organizations DEFAULT VALUES;
INSERT INTO devices (serial_number, status) VALUES ('device-001', 'active');
INSERT INTO vehicles (organization_id, vin, number_plate) VALUES (1, 'VIN123', 'ABC-001');
INSERT INTO device_assignments (vehicle_id, device_id) VALUES (1, 1);
```

### View Telemetry Data

```bash
docker exec -it fleettrack-postgres-1 psql -U postgres -d fleettrack \
  -c "SELECT * FROM telemetry ORDER BY created_at DESC LIMIT 10;"
```

## Useful Commands

```bash
make run              # Run locally
make docker-up        # Start Docker containers
make docker-down      # Stop containers
make docker-logs      # View logs
make db               # Connect to PostgreSQL
make fmt              # Format code
make vet              # Vet code
make lint             # Run linter
```

## Configuration

Environment variables (`.env`):
- `DB_USER` — PostgreSQL username
- `DB_PASSWORD` — PostgreSQL password
- `DB_NAME` — Database name
- `DB_PORT` — PostgreSQL port (5433 external, 5432 internal)
- `API_PORT` — API server port

## Architecture

**Layers:**
1. **Handler** — HTTP request/response handling, error mapping
2. **Service** — Business logic, validation, orchestration
3. **Repository** — Data access abstraction
4. **Model** — Domain models and error definitions

**Key Principles:**
- Dependency Inversion — Service depends on interfaces, not implementations
- Single Responsibility — Each layer has one reason to change
- Clean separation of concerns

## Development

### Running Tests
```bash
go test ./...
```

### Building Binary
```bash
make build
```