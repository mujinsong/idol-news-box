.PHONY: build run clean test tidy

APP_NAME := weibo-spider
BUILD_DIR := ./bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/weibo-spider

run:
	go run ./cmd/weibo-spider -config=configs/config.json

clean:
	rm -rf $(BUILD_DIR)

test:
	go test -v ./...

tidy:
	go mod tidy
