## run: run the example (requires MySQL — see infra-up)
.PHONY: run
run:
	go run .

## infra-up: start the MySQL service this example needs (run from the repo root's docker-compose.yml)
.PHONY: infra-up
infra-up:
	docker compose -f ../../docker-compose.yml up -d mysql

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
