## run: run the example (picks up default.pgo automatically via -pgo=auto)
.PHONY: run
run:
	go run .

## run-nopgo: run with PGO explicitly disabled, for contrast
.PHONY: run-nopgo
run-nopgo:
	go run -pgo=off .

## profile: regenerate default.pgo from a representative run
.PHONY: profile
profile:
	go run -pgo=off . -cpuprofile=default.pgo

## compare: benchmark with PGO off vs on (numbers are machine-dependent)
.PHONY: compare
compare:
	@echo "== PGO off =="
	go test -pgo=off -bench=. -run=^$$ -count=3 .
	@echo "== PGO on (default.pgo) =="
	go test -bench=. -run=^$$ -count=3 .

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
