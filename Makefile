.PHONY: run lint fmt test

run:
	go run cmd/main.go

lint:
	golangci-lint run

fmt:
	golangci-lint fmt

test:
	go test ./...
