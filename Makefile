.PHONY: tools build run install install-sync fmt lint test test-race check coverage coverage-html docs-build docs-serve clean version demo-gif

GO := $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)
UV := $(shell command -v uv 2>/dev/null || echo uv)
GO_BIN_DIR := $(patsubst %/,%,$(dir $(GO)))
GO_TOOL_BIN := $(shell /bin/sh -c 'gobin="$$( $(GO) env GOBIN )"; if [ -n "$$gobin" ]; then printf "%s" "$$gobin"; else printf "%s/bin" "$$( $(GO) env GOPATH )"; fi')
PYTHON_USER_BIN := $(HOME)/.local/bin
export PATH := $(GO_BIN_DIR):$(GO_TOOL_BIN):$(PYTHON_USER_BIN):$(PATH)

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
BUILDINFO_PKG := atbuy/noteui/internal/buildinfo
OUTBIN := noteui
SYNCBIN := noteui-sync

LDFLAGS := '-X $(BUILDINFO_PKG).Version=$(VERSION)'

tools:
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install github.com/incu6us/goimports-reviser/v3@latest
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	$(UV) tool install --upgrade pre-commit
	pre-commit install

build:
	$(GO) build -p 12 -o ./bin/$(OUTBIN) -ldflags=$(LDFLAGS) ./cmd/$(OUTBIN)
	$(GO) build -p 12 -o ./bin/$(SYNCBIN) -ldflags=$(LDFLAGS) ./cmd/$(SYNCBIN)

run: build
	./bin/$(OUTBIN)

install:
	$(GO) install -ldflags=$(LDFLAGS) ./cmd/$(OUTBIN)
	$(GO) install -ldflags=$(LDFLAGS) ./cmd/$(SYNCBIN)

install-sync:
	$(GO) install -ldflags=$(LDFLAGS) ./cmd/$(SYNCBIN)

fmt:
	goimports-reviser -rm-unused -recursive -project-name atbuy/noteui ./cmd ./internal
	gofumpt -w ./cmd ./internal

lint:
	golangci-lint run --config .golangci.yml ./...

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

check:
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) test-race
	$(GO) build ./...

coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -n 1

coverage-html:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

docs-build:
	uvx zensical build --clean

docs-serve:
	uvx zensical serve

clean:
	rm -rf ./bin/$(OUTBIN) ./bin/$(SYNCBIN)

demo-gif: build
	cp ./bin/$(OUTBIN) /tmp/noteui-demo && PATH="/tmp:$$PATH" TERM=xterm-256color vhs demo/demo.tape

version:
	@echo $(VERSION)
