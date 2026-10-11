BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_VERSION = $(shell git describe --tags --always 2>/dev/null || echo dev)
MARKETING_VERSION = $(shell cat MARKETING_VERSION)

.PHONY: all build install test clean

all: build

build:
	go build -ldflags "-X main.buildTime=$(BUILD_TIME) -X main.buildVersion=$(BUILD_VERSION) -X main.marketingVersion=$(MARKETING_VERSION)" -o build/warchiver ./cmd/warchiver

install:
	@if [ -x build/warchiver ]; then \
		dest="$$(go env GOBIN)"; \
		if [ -z "$$dest" ]; then dest="$$(go env GOPATH | cut -d: -f1)/bin"; fi; \
		mkdir -p "$$dest"; \
		cp build/warchiver "$$dest/warchiver"; \
		echo "Installed build/warchiver to $$dest/warchiver"; \
	else \
		go install ./cmd/warchiver; \
	fi

test:
	go test ./...

clean:
	rm -rf build/
