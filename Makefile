.PHONY: build test vet clean install lint

BINARY := rsm
BUILD_DIR := ./dist
CMD := ./cmd/rsm

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(CMD)

install:
	go install $(CMD)

test:
	go test ./... -v

test-short:
	go test ./... -short

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

lint:
	golangci-lint run ./...

.DEFAULT_GOAL := build
