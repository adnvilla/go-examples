## run: run the example (requires Postgres: make infra-up)
.PHONY: run
run:
	go run .

## infra-up: start Postgres (run from the repo root's docker-compose.yml)
.PHONY: infra-up
infra-up:
	docker compose -f ../../docker-compose.yml up -d postgres

## test: run the integration tests against local Postgres (requires infra-up)
.PHONY: test
test:
	POSTGRES_LOCAL=1 go test -race -count=1 -v ./...

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
