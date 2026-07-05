## test: run tests with the race detector (no Test* functions here — see `make bench`)
.PHONY: test
test:
	go test -race -count=1 ./...

## bench: run the benchmarks (requires the docker-compose mysql service running)
.PHONY: bench
bench:
	go test -bench=. -benchtime=1x ./...

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
