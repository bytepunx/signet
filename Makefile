BINARY_DIR := bin
SIGNETD    := $(BINARY_DIR)/signetd
SIGNET     := $(BINARY_DIR)/signet

.PHONY: all build build-tpm proto test test-int lint clean

all: build

build:
	mkdir -p $(BINARY_DIR)
	go build -o $(SIGNETD) ./cmd/signetd
	go build -o $(SIGNET) ./cmd/signet

build-tpm:
	mkdir -p $(BINARY_DIR)
	go build -tags tpm -o $(SIGNETD) ./cmd/signetd
	go build -o $(SIGNET) ./cmd/signet

proto:
	buf generate

test:
	go test ./...

test-int:
	go test -tags integration ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BINARY_DIR)
