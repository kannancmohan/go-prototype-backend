# include .envrc

.PHONY: tidy gofmt build run test test-skip-integration-tests coverage lint

tidy:
	@go mod tidy

gofmt: tidy
	@find . -type f -name '*.go' -not -path './vendor/*' -not -path './pkg/mod/*' -exec gofmt -s -w {} +

build:
	@go build -o bin/api cmd/api/*.go

run:
	@go run cmd/api/*.go

test:
	@go test -v ./...

test-skip-integration-tests:
	@go test -v -tags skip_integration_tests ./...

coverage:
	@rm -f coverage.out && \
	go test -coverprofile=coverage.out ./... && \
	go tool cover -html=coverage.out

lint: tidy gofmt
	@golangci-lint run ./... -v