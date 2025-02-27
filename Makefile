GO           := go
GOFMT        := $(GO)fmt
VERSION      := $(shell cat VERSION 2>/dev/null || echo "development")
COMMIT       := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE   := $(shell date -u +"%Y-%m-%d" 2>/dev/null)
LDFLAGS      := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)
BINARY_NAME  := filewatch_exporter

.PHONY: all
all: style build

.PHONY: style
style:
	$(GOFMT) -l -w .

.PHONY: build
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/filewatch_exporter

.PHONY: test
test:
	$(GO) test -v ./...

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)

.PHONY: run
run: build
	./$(BINARY_NAME) --config=config/config.yaml