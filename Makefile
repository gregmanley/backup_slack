# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Project parameters
BINARY_NAME=backup_slack
BINARY_UNIX=$(BINARY_NAME)_unix
BIN_DIR=bin

# Shell specification
SHELL=/bin/bash

.PHONY: all build run clean test deps build-linux

all: test build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) -v ./cmd/$(BINARY_NAME)

test:
	$(GOTEST) -v ./...

clean:
	@echo "Cleaning up..."
	$(GOCLEAN)
	rm -f $(BIN_DIR)/$(BINARY_NAME)
	rm -f $(BIN_DIR)/$(BINARY_UNIX)

run: build
	@echo "Running $(BINARY_NAME)..."
	@if [ -f .env ]; then 		source .env && 		$(BIN_DIR)/$(BINARY_NAME); 	else 		echo ".env file not found"; 		exit 1; 	fi

deps:
	$(GOMOD) tidy

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BIN_DIR)/$(BINARY_UNIX) -v ./cmd/$(BINARY_NAME)
