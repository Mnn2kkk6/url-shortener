.PHONY: run build test tidy docker-up docker-down docker-logs

run:
	go run ./cmd/server

build:
	go build -o bin/urlshortener ./cmd/server

test:
	go test ./... -v

tidy:
	go mod tidy

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
