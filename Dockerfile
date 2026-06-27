# build stage
FROM golang:1.26.3 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o fleettrack-api ./cmd/api

#runtime stage
FROM alpine
WORKDIR /app:1.26.3

COPY --from=builder /app/fleettrack-api .
EXPOSE 8080

CMD ["./fleettrack-api"]