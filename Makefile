SHELL := /bin/bash
.DEFAULT_GOAL := help

## help: print this help message
.PHONY: help
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: compile every workspace module (root + each migrated example)
.PHONY: build
build:
	@for d in $$(go list -m -f '{{.Dir}}'); do \
		echo "==> build $$d"; \
		(cd "$$d" && go build ./...) || exit 1; \
	done

## test: run all tests with the race detector, in every workspace module
.PHONY: test
test:
	@for d in $$(go list -m -f '{{.Dir}}'); do \
		echo "==> test $$d"; \
		(cd "$$d" && go test -race -count=1 ./...) || exit 1; \
	done

## vet: run go vet in every workspace module
.PHONY: vet
vet:
	@for d in $$(go list -m -f '{{.Dir}}'); do \
		echo "==> vet $$d"; \
		(cd "$$d" && go vet ./...) || exit 1; \
	done

## lint: run golangci-lint in every workspace module (requires golangci-lint in PATH)
.PHONY: lint
lint:
	@for d in $$(go list -m -f '{{.Dir}}'); do \
		echo "==> lint $$d"; \
		(cd "$$d" && golangci-lint run --timeout=5m ./...) || exit 1; \
	done

## vuln: run govulncheck in every workspace module (requires govulncheck in PATH)
.PHONY: vuln
vuln:
	@for d in $$(go list -m -f '{{.Dir}}'); do \
		echo "==> vuln $$d"; \
		(cd "$$d" && govulncheck ./...) || exit 1; \
	done

## tidy: tidy and verify go.mod / go.sum in every workspace module
.PHONY: tidy
tidy:
	@for d in $$(go list -m -f '{{.Dir}}'); do \
		echo "==> tidy $$d"; \
		(cd "$$d" && go mod tidy && go mod verify) || exit 1; \
	done

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
