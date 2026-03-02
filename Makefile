.PHONY: build build-all release clean

PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -o bin/1688-mcp-darwin-arm64 .
	go build -o bin/1688-login-darwin-arm64 ./cmd/login

build-all:
	@mkdir -p bin
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build -o bin/1688-mcp-$$os-$$arch$$ext . ; \
		GOOS=$$os GOARCH=$$arch go build -o bin/1688-login-$$os-$$arch$$ext ./cmd/login ; \
	done

release: build-all
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		name="1688-mcp-$$os-$$arch"; \
		mkdir -p dist/$$name; \
		cp scripts/1688_client.py dist/$$name/; \
		if [ "$$os" = "windows" ]; then \
			cp bin/1688-mcp-$$os-$$arch.exe dist/$$name/; \
			cp bin/1688-login-$$os-$$arch.exe dist/$$name/; \
			cd dist && zip -r $$name.zip $$name && cd ..; \
		else \
			cp bin/1688-mcp-$$os-$$arch dist/$$name/; \
			cp bin/1688-login-$$os-$$arch dist/$$name/; \
			cd dist && tar -czf $$name.tar.gz $$name && cd ..; \
		fi; \
		rm -rf dist/$$name; \
		echo "Packed dist/$$name"; \
	done

clean:
	rm -f bin/1688-mcp-* bin/1688-login-*
	rm -rf dist/
