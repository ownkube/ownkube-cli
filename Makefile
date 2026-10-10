BINARY    := okctl
MODULE    := github.com/ownkube/okctl
VERSION   ?= dev
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE      := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build install test lint clean generate

build:
	go build -ldflags '$(LDFLAGS)' -o bin/$(BINARY) .

install:
	go install -ldflags '$(LDFLAGS)' .

test:
	go test ./...

lint:
	golangci-lint run

# SPEC_SRC: optional path to an updated spec, copied over api/openapi.json first.
generate:
ifdef SPEC_SRC
	cp $(SPEC_SRC) api/openapi.json
endif
	oapi-codegen -config oapi-codegen.yaml api/openapi.json

clean:
	rm -rf bin/ dist/
