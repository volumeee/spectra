.PHONY: build build-cli build-plugins build-all test lint run docker-build docker-up clean

BINARY=spectra
CLI_BINARY=spectra-cli
PLUGIN_DIR=plugins
BUILD_DIR=bin

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/spectra/

build-cli:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(CLI_BINARY) ./cmd/spectra-cli/

build-plugins:
	@mkdir -p $(BUILD_DIR)/plugins
	@for dir in $(PLUGIN_DIR)/*/; do \
		name=$$(basename $$dir); \
		echo "Building plugin: $$name"; \
		go build -o $(BUILD_DIR)/plugins/$$name $$dir; \
		if [ -f "$$dir/plugin.json" ]; then \
			cp "$$dir/plugin.json" "$(BUILD_DIR)/plugins/$${name}.json"; \
		fi; \
	done

build-all: build build-cli build-plugins

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

run: build build-plugins
	./$(BUILD_DIR)/$(BINARY)

docker-build:
	docker build -f docker/Dockerfile -t spectra:latest .

docker-up:
	docker compose -f docker/docker-compose.yml up --build

clean:
	rm -rf $(BUILD_DIR)
