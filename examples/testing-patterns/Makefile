## run: run the example
.PHONY: run
run:
	go run .

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

## fuzz: run the fuzzer for 30s (usage: make fuzz FUZZTIME=10s to override)
.PHONY: fuzz
fuzz:
	go test -fuzz=FuzzAdd -fuzztime=$${FUZZTIME:-30s} .
