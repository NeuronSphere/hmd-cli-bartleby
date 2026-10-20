BINARY    := bartleby
GO_DIR    := src/go/bartleby
REQTRACE_DIR := src/go/reqtrace
BUILD_DIR := $(GO_DIR)/build

# Two Go modules live here: the CLI, and reqtrace, which is carved out under
# Apache-2.0 so projects that cannot take a BSL dependency can consume it.
# Anything that checks sources has to walk both, or the carve-out quietly stops
# being tested while its tests still count as requirement coverage.
MODULES   := $(GO_DIR) $(REQTRACE_DIR)

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
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		( cd $$m && $(GO) test $(GOFLAGS) ./... ) || exit 1; \
	done

## test-verbose: run the Go unit tests with verbose output
test-verbose:
	cd $(GO_DIR) && $(GO) test $(GOFLAGS) -v ./...

## test-race: run the Go unit tests under the race detector
test-race:
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		( cd $$m && $(GO) test $(GOFLAGS) -race -count=1 ./... ) || exit 1; \
	done

## cover: report unit-test coverage per package
cover:
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		( cd $$m && $(GO) test $(GOFLAGS) -cover ./... ) || exit 1; \
	done

## vet: run go vet
vet:
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		( cd $$m && $(GO) vet ./... ) || exit 1; \
	done

## fmt: format the Go sources
fmt:
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		( cd $$m && gofmt -w . ) || exit 1; \
	done

## fmt-check: fail if any Go source is unformatted
fmt-check:
	@for m in $(MODULES); do \
		files=$$(cd $$m && gofmt -l .); \
		if [ -n "$$files" ]; then \
			echo "unformatted files in $$m (run make fmt):"; echo "$$files"; exit 1; \
		fi; \
	done

## reqs: regenerate docs/requirements/traceability.rst from the requirements and test annotations
reqs:
	cd $(REQTRACE_DIR) && $(GO) run ./cmd/reqtrace -repo $(CURDIR)

## reqs-check: fail if traceability has a gap or the generated matrix is stale
reqs-check:
	cd $(REQTRACE_DIR) && $(GO) run ./cmd/reqtrace -repo $(CURDIR) -check

## check: fmt-check + vet + unit tests + traceability — what CI should run
check: fmt-check vet test reqs-check

## tidy: tidy and verify Go modules
tidy:
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		( cd $$m && $(GO) mod tidy && $(GO) mod verify ) || exit 1; \
	done

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
