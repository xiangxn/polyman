APP=polyman
PKG=github.com/xiangxn/polyman/internal/version

VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS = -s -w \
	-X '$(PKG).Version=$(VERSION)' \
	-X '$(PKG).Commit=$(COMMIT)' \
	-X '$(PKG).Date=$(DATE)'

.PHONY: build-linux build-mac clean

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP) ./cmd/polyman

build-mac:
	go build -o $(APP) ./cmd/polyman

clean:
	rm -f $(APP)
