## run: run the example (requires Redis: make infra-up)
.PHONY: run
run:
	go run .

## infra-up: start Redis (run from the repo root's docker-compose.yml)
.PHONY: infra-up
infra-up:
	docker compose -f ../../docker-compose.yml up -d redis

## test: run the integration tests against local Redis (requires infra-up)
.PHONY: test
test:
	REDIS_LOCAL=1 go test -race -count=1 -v ./...

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
