# Load variables from .env file
include .env
export $(shell sed 's/=.*//' .env)

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

## lint: Run golangci-lint on the codebase
.PHONY: lint
lint:
	golangci-lint run

## tidy: tidy module dependency and format all .go files
.PHONY: tidy
tidy:
	@echo 'Tidying module dependencies...'
	go mod tidy
	@echo 'Verifying and vendoring module dependencies'
	go mod verify
	go mod vendor
	@echo 'Formatting .go files...'
	go fmt ./...

# audit run quality control checks
.PHONY: audit
audit:
	@echo 'Checking module dependencies'
	go mod tidy -diff
	go mod verify
	@echo 'Vetting code...'
	go vet ./...
	go tool staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

.PHONY: build
build:
	@echo "Building cmd/api..."
	go build -ldflags='-s' -o=./bin/api ./cmd/api

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	go run ./cmd/api