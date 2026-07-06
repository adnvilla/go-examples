## run: run the example (server + client in one process)
.PHONY: run
run:
	go run .

## test: run tests with the race detector (uses bufconn, no network)
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

## generate: regenerate greeterpb/ from greeter.proto (requires protoc, protoc-gen-go, protoc-gen-go-grpc)
.PHONY: generate
generate:
	protoc -I=./ --go_out=./greeterpb --go_opt=paths=source_relative \
		--go-grpc_out=./greeterpb --go-grpc_opt=paths=source_relative \
		./greeter.proto
