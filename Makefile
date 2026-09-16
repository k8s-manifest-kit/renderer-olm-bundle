MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
LOCAL_BIN_PATH := ${PROJECT_PATH}/bin

LINT_GOGC := 10
LINT_TIMEOUT := 10m

## Tools
GOLANGCI_VERSION ?= v2.12.2
GOLANGCI ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
GOVULNCHECK_VERSION ?= latest
GOVULNCHECK ?= go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

ifndef ignore-not-found
  ignore-not-found = false
endif

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: test

.PHONY: clean
clean:
	go clean -x
	go clean -x -testcache

.PHONY: fmt
fmt:
	@$(GOLANGCI) fmt --config .golangci.yml ./pkg/...
	go fmt ./...

.PHONY: test
test:
	go test -v ./...

.PHONY: bench
bench:
	go test -bench=. -benchmem ./...

.PHONY: deps
deps:
	go mod tidy

.PHONY: deps/update-internal
deps/update-internal:
	@for mod in $$(go list -m -f '{{if and (not .Main) (not .Indirect)}}{{.Path}}{{end}}' all); do \
		case "$$mod" in \
			github.com/k8s-manifest-kit/*) go get "$$mod@main" ;; \
		esac; \
	done
	go mod tidy

.PHONY: deps/update-gomega-matchers
deps/update-gomega-matchers:
	@go get github.com/lburgazzoli/gomega-matchers@main
	go mod tidy

.PHONY: deps/update-direct
deps/update-direct:
	@while read -r mod version; do \
		case "$$mod" in \
			"") ;; \
			github.com/k8s-manifest-kit/*) ;; \
			github.com/lburgazzoli/gomega-matchers) ;; \
			*) go get "$$mod@$$version" ;; \
		esac; \
	done < <(go list -m -u -f '{{if and (not .Main) (not .Indirect) .Update}}{{.Path}} {{.Update.Version}}{{end}}' all)
	go mod tidy

.PHONY: deps/update
deps/update: deps/update-internal deps/update-gomega-matchers deps/update-direct

.PHONY: lint
lint:
	@$(GOLANGCI) run --config .golangci.yml --timeout $(LINT_TIMEOUT) ./pkg/...

.PHONY: lint/fix
lint/fix:
	@$(GOLANGCI) run --config .golangci.yml --timeout $(LINT_TIMEOUT) --fix ./pkg/...

.PHONY: vulncheck
vulncheck:
	@$(GOVULNCHECK) ./...

.PHONY: check
check: lint vulncheck

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)
