BINARY := gh-repos-hud
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test lint vuln run serve install clean tidy

build: ## Build the gh extension binary
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) .

test: ## Run tests with the race detector
	go test -race -count=1 ./...

lint: ## Vet + golangci-lint
	go vet ./...
	golangci-lint run

vuln: ## Scan for known vulnerabilities
	govulncheck ./...

run: build ## Build then launch the TUI
	./$(BINARY)

serve: build ## Build then start the local web dashboard
	./$(BINARY) serve

install: build ## Install as a local gh extension (gh repos-hud)
	gh extension install .

tidy: ## Tidy go.mod / go.sum
	go mod tidy

clean:
	rm -f $(BINARY)
	rm -rf dist
