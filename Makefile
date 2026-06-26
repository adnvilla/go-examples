SHELL := /bin/bash
.DEFAULT_GOAL := help

## help: print this help message
.PHONY: help
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: compile all examples
.PHONY: build
build:
	go build ./...

## test: run all tests with the race detector
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

## vuln: run govulncheck (requires govulncheck in PATH)
.PHONY: vuln
vuln:
	govulncheck ./...

## tidy: tidy and verify go.mod / go.sum
.PHONY: tidy
tidy:
	go mod tidy
	go mod verify

## run: run a specific example  (usage: make run EXAMPLE=context)
.PHONY: run
run:
	@if [ -z "$(EXAMPLE)" ]; then \
		echo "Usage: make run EXAMPLE=<name>  (e.g. make run EXAMPLE=context)"; \
		exit 1; \
	fi
	go run ./examples/$(EXAMPLE)/

## infra-up: start all Docker Compose services
.PHONY: infra-up
infra-up:
	docker compose up -d

## infra-down: stop and remove all Docker Compose services
.PHONY: infra-down
infra-down:
	docker compose down

## ci: run the full local CI pipeline (build + vet + test + lint)
.PHONY: ci
ci: build vet test lint
