## infra-up: start DynamoDB Local (run from the repo root's docker-compose.yml)
.PHONY: infra-up
infra-up:
	docker compose -f ../../docker-compose.yml up -d dynamodb

## test: run the CRUD integration tests against DynamoDB Local (requires infra-up)
.PHONY: test
test:
	DYNAMODB_LOCAL=1 go test -race -count=1 -v ./...

## run: this example has no main-package logic to run; see `make test` for the demonstration
.PHONY: run
run:
	go run .

## vet: run go vet
.PHONY: vet
vet:
	go vet ./...

## lint: run golangci-lint (requires golangci-lint in PATH)
.PHONY: lint
lint:
	golangci-lint run ./...

## fmt: check gofmt formatting
.PHONY: fmt
fmt:
	@test -z "$$(gofmt -l .)" || (echo "not gofmt'd:"; gofmt -l .; exit 1)
