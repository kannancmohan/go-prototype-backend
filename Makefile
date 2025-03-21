# include .envrc

.PHONY: tidy gofmt build run test test-skip-integration-tests lint

tidy:
	@go mod tidy

gofmt:
	@find . -type f -name '*.go' -not -path './vendor/*' -not -path './pkg/mod/*' -exec gofmt -s -w {} +

build:
	@go build -o bin/api cmd/api/*.go

run:
	@go run cmd/api/*.go

test:
	@go test -v ./...

test-skip-integration-tests: gogenerate
	@go test -v -tags skip_integration_tests ./...

lint: tidy gofmt
	@golangci-lint run ./... -v
	@go vet ./...