GO := go
BINARY_NAME := meilisearch-proxy
BIN_DIR := bin

.PHONY: run build

install-deps:


run: build
	$(BIN_DIR)/$(BINARY_NAME)

build:
	$(GO) build -o $(BIN_DIR)/$(BINARY_NAME) cmd/meilisearch-proxy/main.go

docker-build:
	docker build -t registry.maxroll.gg/library/meilisearch-proxy .

test:
	ginkgo -r -v