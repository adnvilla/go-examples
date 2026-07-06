## run: run the example (spans print to stdout; no infra needed)
.PHONY: run
run:
	go run .

## run-jaeger: run the example exporting to Jaeger (requires infra-up)
.PHONY: run-jaeger
run-jaeger:
	OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run .

## infra-up: start Jaeger (run from the repo root's docker-compose.yml)
.PHONY: infra-up
infra-up:
	docker compose -f ../../docker-compose.yml up -d jaeger

## test: run tests with the race detector
.PHONY: test
test:
	go test -race -count=1 ./...

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
