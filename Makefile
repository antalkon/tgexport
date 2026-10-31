BINARY  := tgexport
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: all build install test cover lint fmt golden snapshot clean

all: lint test build

build: ## Build ./bin/tgexport
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/tgexport

install: ## Install into $GOBIN
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/tgexport

test: ## Run tests with the race detector
	go test -race -count=1 ./...

cover: ## Show coverage per function
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

lint: ## gofmt, go vet, golangci-lint (if installed)
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	go vet ./...
	@command -v golangci-lint >/dev/null && golangci-lint run || echo "golangci-lint not installed, skipped"

fmt: ## Format code
	gofmt -w .

golden: ## Regenerate golden files after an intended output change
	go test ./internal/export/ -run TestGolden -update

snapshot: ## Local cross-platform build via GoReleaser (no publish)
	goreleaser release --snapshot --clean

clean:
	rm -rf bin dist coverage.out
