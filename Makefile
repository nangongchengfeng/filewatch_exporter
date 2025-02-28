# Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
BINARY_NAME=filewatch_exporter
BINARY_UNIX=$(BINARY_NAME)
MAIN_PATH=./

# Build-time variables
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

.PHONY: all build clean run build-linux

all: build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) $(MAIN_PATH)

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

run: build
	./$(BINARY_NAME) -config=config/config.yaml

# 交叉编译 Windows 到 Linux (在 Windows 上运行)
build-linux-windows:
	SET CGO_ENABLED=0&&SET GOOS=linux&&SET GOARCH=amd64&& $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) $(MAIN_PATH)
