.PHONY: build build-all clean

build:
	go build -o bin/1688-mcp-darwin-arm64 .
	go build -o bin/1688-login-darwin-arm64 ./cmd/login

build-all:
	GOOS=darwin GOARCH=arm64 go build -o bin/1688-mcp-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o bin/1688-mcp-darwin-amd64 .
	GOOS=linux  GOARCH=amd64 go build -o bin/1688-mcp-linux-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o bin/1688-login-darwin-arm64 ./cmd/login
	GOOS=darwin GOARCH=amd64 go build -o bin/1688-login-darwin-amd64 ./cmd/login
	GOOS=linux  GOARCH=amd64 go build -o bin/1688-login-linux-amd64 ./cmd/login

clean:
	rm -f bin/1688-mcp-* bin/1688-login-*
