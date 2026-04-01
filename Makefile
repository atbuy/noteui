.PHONY: build run install test docs-build docs-serve clean version

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
BUILDINFO_PKG := atbuy/noteui/internal/buildinfo
OUTBIN := noteui

LDFLAGS := '-X $(BUILDINFO_PKG).Version=$(VERSION)'

build:
	go build -p 12 -o ./bin/$(OUTBIN) -ldflags=$(LDFLAGS) ./cmd/*

run: build
	./bin/$(OUTBIN)

install:
	go install -ldflags=$(LDFLAGS) ./cmd/$(OUTBIN)

test:
	go test ./...

docs-build:
	uvx zensical build --clean

docs-serve:
	uvx zensical serve

clean:
	rm -rf ./bin/$(OUTBIN)

version:
	@echo $(VERSION)
