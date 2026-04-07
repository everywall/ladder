GO ?= go
BINARY ?= ladder
CMD ?= ./cmd/main.go
BIN_DIR ?= bin
OUT ?= $(BIN_DIR)/$(BINARY)
PKGS ?= ./...
GOFUMPT ?= gofumpt
GOLANGCI_LINT ?= golangci-lint
GOLANGCI_CONFIG ?= .golangci-lint.yaml

.PHONY: help build build-linux run test vet fmt fmt-check lint lint-fix tidy clean check-tools check-fmt-tool check-lint-tool install-tools install-linters

help:
	@echo "Available targets:"
	@echo "  make build          Build binary to $(OUT)"
	@echo "  make build-linux    Cross-compile linux/amd64 binary to $(OUT)-linux-amd64"
	@echo "  make run            Run app with go run"
	@echo "  make test           Run unit tests"
	@echo "  make vet            Run go vet"
	@echo "  make fmt            Format code with gofumpt"
	@echo "  make fmt-check      Check formatting without rewriting files"
	@echo "  make lint           Run formatting check + golangci-lint"
	@echo "  make lint-fix       Auto-fix formatting and lint issues where possible"
	@echo "  make tidy           Run go mod tidy"
	@echo "  make clean          Clean build artifacts"
	@echo "  make install-tools  Install required lint/format tools"

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(OUT) $(CMD)

build-linux:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -o $(OUT)-linux-amd64 $(CMD)

run:
	$(GO) run $(CMD)

test:
	$(GO) test $(PKGS)

vet:
	$(GO) vet $(PKGS)

fmt: check-fmt-tool
	$(GOFUMPT) -w .

fmt-check: check-fmt-tool
	$(GOFUMPT) -l .

lint: check-tools fmt-check
	$(GOLANGCI_LINT) run -c $(GOLANGCI_CONFIG)

lint-fix: check-tools
	$(GOFUMPT) -w .
	$(GOLANGCI_LINT) run -c $(GOLANGCI_CONFIG) --fix

tidy:
	$(GO) mod tidy

clean:
	$(GO) clean
	@rm -rf $(BIN_DIR)

check-tools:
	@$(MAKE) check-fmt-tool
	@$(MAKE) check-lint-tool

check-fmt-tool:
	@command -v $(GOFUMPT) >/dev/null 2>&1 || (echo "missing tool: $(GOFUMPT)" && exit 1)

check-lint-tool:
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || (echo "missing tool: $(GOLANGCI_LINT)" && exit 1)

install-tools:
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2

install-linters: install-tools