.PHONY: build run install install-sync fmt test coverage coverage-html docs-build docs-serve clean version demo-gif

GO := $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)
GO_BIN_DIR := $(patsubst %/,%,$(dir $(GO)))
export PATH := $(GO_BIN_DIR):$(PATH)

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
BUILDINFO_PKG := atbuy/noteui/internal/buildinfo
OUTBIN := noteui
SYNCBIN := noteui-sync

LDFLAGS := '-X $(BUILDINFO_PKG).Version=$(VERSION)'

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

test:
	$(GO) test ./...

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
