lint:
	gofumpt -l -w .
	golangci-lint run
	go mod tidy

install-linters:
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2