# nvy — minimalist runtime version manager
# -------------------------------------------
# Targets:
#   make          — build the binary
#   make install  — install to /usr/local/bin (or GOBIN)
#   make test     — run tests
#   make lint     — run golangci-lint
#   make cover    — run tests and show coverage report
#   make clean    — remove build artefacts

BINARY             := nvy
VERSION            ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOFLAGS            := -trimpath
LDFLAGS            := -s -w -X github.com/trevorphillipscoding/nvy/cmd.Version=$(VERSION)
COVERAGE_THRESHOLD := 65
COVER_PKGS         := ./internal/...,./plugins/...

.PHONY: all build install test lint cover cover-check tidy clean deps help

all: build

## build: compile the nvy binary into the project root
build:
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) .

## install: install nvy to $(GOBIN) or /usr/local/bin
install:
	go install $(GOFLAGS) -ldflags "$(LDFLAGS)" .

## test: run all tests
test:
	go test ./...

## lint: run golangci-lint (install: https://github.com/golangci/golangci-lint)
lint:
	golangci-lint run ./...

# Format: run gofmt
format:
	gofmt -s -w .

## cover: run tests and show coverage report
cover:
	go test -coverprofile=coverage.out -coverpkg=$(COVER_PKGS) ./...
	go tool cover -func=coverage.out

## cover-check: fail if total coverage is below $(COVERAGE_THRESHOLD)%
cover-check:
	go test -coverprofile=coverage.out -coverpkg=$(COVER_PKGS) ./...
	@TOTAL=$$(go tool cover -func=coverage.out | awk '/^total:/{print $$3}' | tr -d '%'); \
	echo "Coverage: $${TOTAL}%"; \
	awk -v t="$${TOTAL}" -v threshold=$(COVERAGE_THRESHOLD) \
	  'BEGIN { if (t+0 < threshold+0) { print "FAIL: coverage " t "% is below required " threshold "%"; exit 1 } \
	           else { print "PASS: coverage " t "% meets threshold " threshold "%" } }'

## tidy: tidy and verify go modules
tidy:
	go mod tidy
	go mod verify

## clean: remove the compiled binary
clean:
	rm -f $(BINARY)

## deps: download dependencies
deps:
	go mod download

## help: print this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
