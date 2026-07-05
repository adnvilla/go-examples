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

## generate: regenerate wire_gen.go from wire.go (requires the wire CLI: go install github.com/google/wire/cmd/wire@latest)
.PHONY: generate
generate:
	wire gen ./...
