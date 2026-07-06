## run: this is a benchmark-only package with no main; see `make bench`
.PHONY: run
run:
	go test -bench=. -run=^$$ ./...

## bench: run the dispatch-strategy benchmarks
.PHONY: bench
bench:
	go test -bench=. -run=^$$ ./...

## test: run tests with the race detector (no Test* functions here — see `make bench`)
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
