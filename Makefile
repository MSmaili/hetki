.PHONY: build install test clean release

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X 'github.com/MSmaili/muxie/cmd.Version=$(VERSION)' \
           -X 'github.com/MSmaili/muxie/cmd.GitCommit=$(COMMIT)' \
           -X 'github.com/MSmaili/muxie/cmd.BuildDate=$(DATE)'

build:
	go build -ldflags "$(LDFLAGS)" -o muxie .

install: build
	sudo mv muxie /usr/local/bin/muxie

test:
	go test -v ./...

clean:
	rm -f muxie
	rm -rf dist/

# Build for multiple platforms (for releases)
release:
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/muxie-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/muxie-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/muxie-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/muxie-linux-arm64 .
	@echo "Release binaries built in dist/"
