SHELL := /bin/bash

.PHONY: deps core-deps ai-deps frontend-deps build test fmt lint compose-up compose-down run-core run-ai run-frontend migrate-core seed

deps: core-deps ai-deps frontend-deps

core-deps:
	cd core && go mod tidy

ai-deps:
	cd ai-service && go mod tidy

frontend-deps:
	cd frontend && npm install

build:
	cd core && go build ./...
	cd ai-service && go build ./...
	cd frontend && npm run build

test:
	cd core && go test ./...
	cd ai-service && go test ./...

fmt:
	cd core && gofmt -w $$(rg -l '' . --glob '*.go')
	cd ai-service && gofmt -w $$(rg -l '' . --glob '*.go')
	cd frontend && npm run lint -- --fix

lint:
	cd core && go test ./...
	cd ai-service && go test ./...
	cd frontend && npm run lint

compose-up:
	docker compose -f deployments/docker-compose.yml up --build -d

compose-down:
	docker compose -f deployments/docker-compose.yml down -v

run-core:
	cd core && go run ./cmd/server

run-ai:
	cd ai-service && go run ./cmd/server

run-frontend:
	cd frontend && npm run dev

migrate-core:
	cd core && go run ./cmd/server migrate

seed:
	cd scripts/seed && go run .
