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

.PHONY: test/rpt
test/rpt:
	go test -race -vet=off -coverprofile=coverage.out ./... 
	@go tool cover -html=coverage.out -o coverage.html
	@xdg-open coverage.html || open coverage.html

.PHONY: build
build:
	@echo "Building cmd/api..."
	go build -ldflags='-s' -o=./bin/api ./cmd/api

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	go run ./cmd/api

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	psql ${PRODUCTS_DB_DSN}

## db/migrations/new name=<file_name>: Create a new database migration
.PHONY: db/migrations/new
db/migrations/new:
	@if [ -z "$(name)" ]; then \
		echo "Usage: make migration name=<file_name>"; \
		exit 1; \
	fi
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}; \

## db/migrations/up: Apply all up database migrations
.PHONY: db/migrations/up
db/migrations/up: confirm
	@if [ -z "$(PRODUCTS_DB_DSN)" ]; then \
		echo "Usage: PRODUCTS_DB_DSN must be set or passed as a flag"; \
		exit 1; \
	fi
	@echo 'Running migrations up...'
	migrate -path=./migrations -database="$(PRODUCTS_DB_DSN)" up

## db/migrations/down: Revert all database migrations
.PHONY: db/migrations/down
db/migrations/down: confirm
	@if [ -z "$(PRODUCTS_DB_DSN)" ]; then \
		echo "Usage: PRODUCTS_DB_DSN must be set or passed as a flag"; \
		exit 1; \
	fi
	@echo 'Running migrations down...'
	migrate -path=./migrations -database="$(PRODUCTS_DB_DSN)" down

## db/migrations/force version=<version>: Force the database schema to a specific version. Does not run any sql (useful for fixing broken migrations)
.PHONY: db/migrations/force
db/migrations/force: confirm
	@if [ -z "$(PRODUCTS_DB_DSN)" ]; then \
		echo "Usage: PRODUCTS_DB_DSN must be set or passed as a flag"; \
		exit 1; \
	fi
	@echo 'Running force migrations...'
	migrate -path=./migrations -database="$(PRODUCTS_DB_DSN)" force ${version}
