## run: attempt to run the handler locally — fails fast with a clear error (see README); this is expected
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

## build-lambda: cross-compile for AWS Lambda's runtime (linux/amd64)
.PHONY: build-lambda
build-lambda:
	GOOS=linux GOARCH=amd64 go build -o bootstrap .
	zip main.zip bootstrap
	rm bootstrap
