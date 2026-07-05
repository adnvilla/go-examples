## run: run the example (builds a 10,000,000-element slice via reflection)
.PHONY: run
run:
	go run .

## test: run tests with the race detector (Benchmark* functions here are not Test* — see `make bench`)
.PHONY: test
test:
	go test -race -count=1 ./...

## bench: compare plain append vs. reflect.Append for a single slice build
.PHONY: bench
bench:
	go test -bench='BenchmarkSlice$$|BenchmarkSliceReflect$$' -benchtime=1x -run=^$$ ./...

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
