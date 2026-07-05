SHELL := /bin/bash
.DEFAULT_GOAL := help

# If EXAMPLE is set, operate on that single example only (short name, e.g. EXAMPLE=http-server);
# otherwise operate on every workspace module.
# Usage: make build EXAMPLE=http-server   (omit EXAMPLE to run across the whole workspace)
MODULES := $(if $(EXAMPLE),examples/$(EXAMPLE),$(shell go list -m -f '{{.Dir}}'))

## help: print this help message
.PHONY: help
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: compile every workspace module, or one with EXAMPLE= (root + each migrated example)
.PHONY: build
build:
	@for d in $(MODULES); do \
		echo "==> build $$d"; \
		(cd "$$d" && go build ./...) || exit 1; \
	done

## test: run all tests with the race detector; every module, or one with EXAMPLE=
.PHONY: test
test:
	@for d in $(MODULES); do \
		echo "==> test $$d"; \
		(cd "$$d" && go test -race -count=1 ./...) || exit 1; \
	done

## vet: run go vet; every module, or one with EXAMPLE=
.PHONY: vet
vet:
	@for d in $(MODULES); do \
		echo "==> vet $$d"; \
		(cd "$$d" && go vet ./...) || exit 1; \
	done

## fmt: check formatting with gofmt; every module, or one with EXAMPLE=
.PHONY: fmt
fmt:
	@for d in $(MODULES); do \
		echo "==> fmt $$d"; \
		out="$$(gofmt -l "$$d")"; \
		if [ -n "$$out" ]; then echo "$$out"; echo "gofmt: files need formatting"; exit 1; fi; \
	done

## lint: run golangci-lint; every module, or one with EXAMPLE= (requires golangci-lint in PATH)
.PHONY: lint
lint:
	@for d in $(MODULES); do \
		echo "==> lint $$d"; \
		(cd "$$d" && golangci-lint run --timeout=5m ./...) || exit 1; \
	done

## vuln: run govulncheck; every module, or one with EXAMPLE= (requires govulncheck in PATH)
.PHONY: vuln
vuln:
	@for d in $(MODULES); do \
		echo "==> vuln $$d"; \
		(cd "$$d" && govulncheck ./...) || exit 1; \
	done

## tidy: tidy and verify go.mod / go.sum; every module, or one with EXAMPLE=
.PHONY: tidy
tidy:
	@for d in $(MODULES); do \
		echo "==> tidy $$d"; \
		(cd "$$d" && go mod tidy && go mod verify) || exit 1; \
	done

## use: register a new example in the workspace (usage: make use EXAMPLE=http-server)
.PHONY: use
use:
	@if [ -z "$(EXAMPLE)" ]; then \
		echo "Usage: make use EXAMPLE=<name>  (e.g. make use EXAMPLE=http-server)"; \
		exit 1; \
	fi
	go work use ./examples/$(EXAMPLE)

## run: run a specific example (usage: make run EXAMPLE=context)
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

## check: build + vet + test + lint + fmt for one example (usage: make check EXAMPLE=http-server)
.PHONY: check
check: build vet test lint fmt

## ci: run the full local CI pipeline across the whole workspace (build + vet + test + lint)
.PHONY: ci
ci: build vet test lint
