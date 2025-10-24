.PHONY: build install clean

BINARY ?= edenred
BUILD_DIR ?= bin

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/edenred

install:
	go install ./cmd/edenred

clean:
	rm -rf $(BUILD_DIR)
