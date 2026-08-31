BINARY    := bartleby
GO_DIR    := src/go/bartleby
BUILD_DIR := $(GO_DIR)/build

GO        := go
GOFLAGS   :=

VERSION   := $(shell cat meta-data/VERSION 2>/dev/null || echo "dev")
LDFLAGS   := -ldflags "-X main.version=$(VERSION)"

# Robot Framework is optional and not a Go dependency. Override ROBOT to run it
# without installing it, e.g.
#   make test-robot ROBOT="uvx --from robotframework robot"
ROBOT         ?= python3 -m robot
ROBOT_RESULTS := test/results
ROBOT_TS      := $(shell date -u +%Y-%m-%dT%H%M%S)
ROBOT_FLAGS    = --variable BINARY:$(abspath $(BUILD_DIR)/$(BINARY)) \
                 --outputdir $(ROBOT_RESULTS) \
                 --output  $(ROBOT_TS)-output.xml \
                 --log     $(ROBOT_TS)-log.html \
                 --report  $(ROBOT_TS)-report.html

.PHONY: all build test test-verbose test-race cover vet fmt fmt-check check tidy clean run reqs reqs-check test-cli test-robot help

all: build

## build: compile the binary to src/go/bartleby/build/bartleby
build:
	@mkdir -p $(BUILD_DIR)
	cd $(GO_DIR) && $(GO) build $(GOFLAGS) $(LDFLAGS) -o build/$(BINARY) .

## test: run the Go unit tests
test:
	cd $(GO_DIR) && $(GO) test $(GOFLAGS) ./...

## test-verbose: run the Go unit tests with verbose output
test-verbose:
	cd $(GO_DIR) && $(GO) test $(GOFLAGS) -v ./...

## test-race: run the Go unit tests under the race detector
test-race:
	cd $(GO_DIR) && $(GO) test $(GOFLAGS) -race -count=1 ./...

## cover: report unit-test coverage per package
cover:
	cd $(GO_DIR) && $(GO) test $(GOFLAGS) -cover ./...

## vet: run go vet
vet:
	cd $(GO_DIR) && $(GO) vet ./...

## fmt: format the Go sources
fmt:
	cd $(GO_DIR) && gofmt -w .

## fmt-check: fail if any Go source is unformatted
fmt-check:
	@cd $(GO_DIR) && files=$$(gofmt -l .); \
	if [ -n "$$files" ]; then \
		echo "unformatted files (run make fmt):"; echo "$$files"; exit 1; \
	fi

## reqs: regenerate docs/requirements/traceability.rst from the requirements and test annotations
reqs:
	cd $(GO_DIR) && $(GO) run ./tools/reqtrace

## reqs-check: fail if traceability has a gap or the generated matrix is stale
reqs-check:
	cd $(GO_DIR) && $(GO) run ./tools/reqtrace -check

## check: fmt-check + vet + unit tests + traceability — what CI should run
check: fmt-check vet test reqs-check

## tidy: tidy and verify Go modules
tidy:
	cd $(GO_DIR) && $(GO) mod tidy && $(GO) mod verify

## clean: remove build artifacts and test output
clean:
	rm -rf $(BUILD_DIR) $(ROBOT_RESULTS)

## run: build and print help (quick smoke test)
run: build
	$(BUILD_DIR)/$(BINARY) --help

## test-cli: build then run the CLI contract tests (fast, no Docker needed)
test-cli: build
	@mkdir -p $(ROBOT_RESULTS)
	$(ROBOT) $(ROBOT_FLAGS) test/cli.robot

## test-robot: build then run every Robot suite (needs Docker + the transform image)
test-robot: build
	@mkdir -p $(ROBOT_RESULTS)
	$(ROBOT) $(ROBOT_FLAGS) \
	      test/cli.robot \
	      test/preconditions.robot \
	      test/bartleby_cli.robot

## help: show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
