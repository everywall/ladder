build:
	cd proxychain/codegen && go run codegen.go
	git submodule update --init --recursive
	git rev-parse --short HEAD > handlers/VERSION
	git rev-parse --short HEAD > cmd/VERSION
	go build -o ladder -ldflags="-s -w" cmd/main.go

lint:
	gofumpt -l -w .
	golangci-lint run -c .golangci-lint.yaml --fix

	go mod tidy
	go clean

install-linters:
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2

run:
	go run ./cmd/.
