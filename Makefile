APP_NAME := agent-cli

.PHONY: build test lint clean release

build:
	go build -o bin/$(APP_NAME) ./cmd/agent-cli

test:
	go test -race ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run

clean:
	rm -rf ./bin ./dist

release:
	go run github.com/goreleaser/goreleaser/v2@latest release --clean
