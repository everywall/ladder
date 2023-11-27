lint:
	gofumpt -l -w .
	golangci-lint run -c .golangci-lint.yaml --fix

	go mod tidy
	go clean

install-linters:
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2