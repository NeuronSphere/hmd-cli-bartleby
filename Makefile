BINARY    := bartleby
GO_DIR    := src/go/bartleby
BUILD_DIR := $(GO_DIR)/build

GO        := go
GOFLAGS   :=

VERSION   := $(shell cat meta-data/VERSION 2>/dev/null || echo "dev")
LDFLAGS   := -ldflags "-X main.version=$(VERSION)"

ROBOT         := python3 -m robot
ROBOT_RESULTS := test/results
ROBOT_TS      := $(shell date -u +%Y-%m-%dT%H%M%S)

.PHONY: all build test test-verbose vet tidy clean run test-robot help

all: build

## build: compile the binary to src/go/bartleby/build/bartleby
build:
	@mkdir -p $(BUILD_DIR)
	cd $(GO_DIR) && $(GO) build $(GOFLAGS) $(LDFLAGS) -o build/$(BINARY) .

## test: run all Go tests
test:
	cd $(GO_DIR) && $(GO) test $(GOFLAGS) ./...

## test-verbose: run all Go tests with verbose output
test-verbose:
	cd $(GO_DIR) && $(GO) test $(GOFLAGS) -v ./...

## vet: run go vet
vet:
	cd $(GO_DIR) && $(GO) vet ./...

## tidy: tidy and verify go modules
tidy:
	cd $(GO_DIR) && $(GO) mod tidy && $(GO) mod verify

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## run: build and print help (quick smoke test)
run: build
	$(BUILD_DIR)/$(BINARY) --help

## test-robot: build then run all Robot Framework tests
test-robot: build
	@mkdir -p $(ROBOT_RESULTS)
	$(ROBOT) --variable BINARY:$(abspath $(BUILD_DIR)/$(BINARY)) \
	      --outputdir $(ROBOT_RESULTS) \
	      --output  $(ROBOT_TS)-output.xml \
	      --log     $(ROBOT_TS)-log.html \
	      --report  $(ROBOT_TS)-report.html \
	      test/preconditions.robot \
	      test/bartleby_cli.robot

## help: show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
