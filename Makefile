.PHONY: build run install clean

OUTBIN=noteui

build:
	go build -p 12 -o ./bin/$(OUTBIN) ./cmd/*

run: build
	./bin/$(OUTBIN)

install:
	go install ./cmd/$(OUTBIN)

clean:
	rm -rf ./bin/$(OUTBIN)
