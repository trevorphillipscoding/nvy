# nvy — minimalist runtime version manager
# -------------------------------------------
# Targets:
#   make          — build the binary
#   make install  — install to /usr/local/bin (or GOBIN)
#   make test     — run tests
#   make clean    — remove build artefacts

BINARY     := nvy
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOFLAGS    := -trimpath
LDFLAGS    := -s -w -X github.com/trevorphillipscoding/nvy/cmd.Version=$(VERSION)

.PHONY: all build install test clean deps

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
