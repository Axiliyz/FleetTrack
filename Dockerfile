# build stage
FROM golang:1.26.3 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o fleettrack-api ./cmd/api

# runtime stage
FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app

COPY --from=builder /app/fleettrack-api .
USER app

EXPOSE 8080 9091

CMD ["./fleettrack-api"]
