BINARY_DIR := bin
SIGNETD    := $(BINARY_DIR)/signetd
SIGNET     := $(BINARY_DIR)/signet

.PHONY: all build build-tpm proto test test-int lint clean

all: build

build: proto
	mkdir -p $(BINARY_DIR)
	go build -o $(SIGNETD) ./cmd/signetd
	go build -o $(SIGNET) ./cmd/signet

build-tpm: proto
	mkdir -p $(BINARY_DIR)
	go build -tags tpm -o $(SIGNETD) ./cmd/signetd
	go build -o $(SIGNET) ./cmd/signet

proto:
	buf generate

test: proto
	go test -race -count=1 ./...

test-int: proto
	go test -race -count=1 -tags integration ./...

lint: proto
	golangci-lint run ./...

clean:
	rm -rf $(BINARY_DIR)
