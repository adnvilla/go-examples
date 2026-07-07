## run: the demonstration lives in the tests; see `make test`
.PHONY: run
run:
	go run .

## test: run the testcontainers demonstration (requires a Docker daemon)
.PHONY: test
test:
	go test -race -count=1 -v ./...

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
