.PHONY: help generate build run test lint vuln tidy up down

help: ## Muestra esta ayuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

generate: ## Regenera el codigo de gqlgen desde el esquema
	go run github.com/99designs/gqlgen generate

build: ## Compila el binario en bin/
	go build -o bin/orders-api ./cmd

run: ## Corre la app localmente
	go run ./cmd

test: ## Corre los tests con race detector y cobertura
	go test ./... -race -covermode=atomic -coverprofile=coverage.out

lint: ## Corre golangci-lint
	golangci-lint run ./...

vuln: ## Corre govulncheck
	govulncheck ./...

tidy: ## Ordena go.mod / go.sum
	go mod tidy

up: ## Levanta app + Postgres con docker compose
	docker compose up --build

down: ## Baja los contenedores
	docker compose down
