BINARY ?= golem
CMD_DIR ?= ./cmd/golem

.PHONY: help build test test-race lint run status chat smoke

help:
	@echo "Targets:"
	@echo "  build      Build golem binary"
	@echo "  test       Run unit tests"
	@echo "  test-race  Run tests with race detector"
	@echo "  lint       Run go vet"
	@echo "  run        Start server mode"
	@echo "  status     Show system status"
	@echo "  chat       Run one-shot ping chat"
	@echo "  smoke      Run phase smoke suite"

build:
	go build -o $(BINARY) $(CMD_DIR)

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	go vet ./...

run:
	go run $(CMD_DIR) run

status:
	go run $(CMD_DIR) status

chat:
	go run $(CMD_DIR) chat "ping"

smoke:
	go test ./...
	go run $(CMD_DIR) status
	go run $(CMD_DIR) chat "ping"
