MODULE  := github.com/abydv/devlab
BINARY  := devlab
BIN_DIR := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.0-dev")
LDFLAGS := -X main.version=$(VERSION)

.PHONY: all
all: fmt vet test build

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) ./cmd/devlab

.PHONY: run
run:
	go run -ldflags "$(LDFLAGS)" ./cmd/devlab

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./...

.PHONY: verify
verify: fmt vet test build

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

.PHONY: tidy
tidy:
	go mod tidy
