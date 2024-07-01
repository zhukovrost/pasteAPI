# Include variables from the .envrc file
include .env

# ==================================================================================== #
# RUN
# ==================================================================================== #

## run/api: run the application
.PHONY: run/api
run/api:
	@echo 'Running app...'
	docker compose up

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## dev/run/api: run the cmd/api application (development only)
.PHONY: dev/run/api
dev/run/api:
	@go run ./cmd/api -db-dsn=${PASTE_DB_DSN} -smtp-password=${PASTE_SMTP_PASSWORD}

## db/psql: connect to the database using psql (development only)
.PHONY: db/psql
db/psql:
	psql ${PASTE_DB_DSN}

## db/migrations/new name=$1: create a new database migration (development only)
.PHONY: db/migrations/new
db/migrations/up:
	@echo 'Running up migrations...'
	migrate -path ./migrations -database ${PASTE_DB_DSN} up

## db/migrations/up: apply all up database migrations (development only)
.PHONY: db/migrations/up
db/migrations/new: confirm
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit: # vendor
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor

# ==================================================================================== #
# BUILD
# ==================================================================================== #

current_time = $(shell date --iso-8601=seconds)
git_description = $(shell git describe --always --dirty --tags --long)
linker_flags = '-s -X pasteAPI/internal/config.Version=${git_description}'
# linker_flags = '-s -X pasteAPI/internal/config.BuildTime=${current_time} -X pasteAPI/internal/config.Version=${git_description}'

## build/api: build the cmd/api application
.PHONY: build/api
build/api: # build/docs
	@echo 'Building cmd/api...'
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64
	go build -ldflags=${linker_flags} -o=./bin/api ./cmd/api

## build/compose: build and deploy using Docker Compose
.PHONY: build/compose
build/compose:
	@echo 'Composing down...'
	docker compose down
	@echo 'Building container...'
	make build/image
	@echo 'Composing up...'
	docker compose up

## build/image: build Docker image for the application
.PHONY: build/image
build/image:
	@echo 'Building container...'
	-docker rmi pasteapi 2>/dev/null || true
	docker build -t pasteapi .

## build/docs: generate API documentation using Swagger
.PHONY: build/docs
build/docs:
	@echo 'Building docs'
	swag init -g ./cmd/api/main.go
