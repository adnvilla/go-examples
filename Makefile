default: build

build:
	go build ./...

test:
	go test -race ./...

vet:
	go vet ./...

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

.PHONY: build test vet tidy lint
