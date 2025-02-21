# include .envrc
# MIGRATIONS_PATH = ./cmd/migrate/migrations

.PHONY: tidy gofmt build gogenerate test test-skip-docker-tests run-api run-indexer lint

tidy:
	@go mod tidy

gofmt:
	@find . -type f -name '*.go' -not -path './vendor/*' -not -path './pkg/mod/*' -exec gofmt -s -w {} +

build:
	@go build -o bin/rest-api cmd/api/*.go
	@go build -o bin/search-indexer cmd/elasticsearch-indexer-kafka/*.go

gogenerate:
	@go generate ./...

test: gogenerate
	@go test -v ./...

# test-skip-docker-tests: gogenerate
# 	@go test -v -tags skip_docker_tests ./...

# run-api: build
# 	@./bin/rest-api

# run-indexer: build
# 	@./bin/search-indexer

lint: tidy gofmt
	@golangci-lint run ./...
	@go vet ./...

# migration-create: 
# 	@migrate create -seq -ext sql -dir $(MIGRATIONS_PATH) $(filter-out $@,$(MAKECMDGOALS))

# migration-up:
# 	@migrate -path=$(MIGRATIONS_PATH) -database=$(DB_ADDR) up

# migration-down:
# 	@migrate -path=$(MIGRATIONS_PATH) -database=$(DB_ADDR) down $(filter-out $@,$(MAKECMDGOALS))