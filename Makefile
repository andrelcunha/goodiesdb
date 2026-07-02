BINARY_NAME=goodiesdb-server
MAIN_PATH=cmd/goodiesdb-server
BUILD_DIR=bin

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

run: build
	@./$(BUILD_DIR)/$(BINARY_NAME)

build:
	@mkdir -p $(BUILD_DIR)
	@VERSION=$$(git describe --tags --always 2>/dev/null || echo dev); \
	go build -ldflags "-X main.version=$$VERSION" -o ./$(BUILD_DIR)/$(BINARY_NAME) ./$(MAIN_PATH)

install:
	@test -x ./$(BUILD_DIR)/$(BINARY_NAME) || { echo "Nothing to install. Run 'make build' first"; exit 1;  }
	@mkdir -p $(BINDIR)
	@install -m 0755 ./$(BUILD_DIR)/$(BINARY_NAME) $(BINDIR)/$(BINARY_NAME)
