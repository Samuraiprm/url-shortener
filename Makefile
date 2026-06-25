.PHONY: up down build run migrate

up:
	docker compose up -d

down:
	docker compose down

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

dev:
	go run ./cmd/server

test:
	go test ./... -v

lint:
	golangci-lint run
