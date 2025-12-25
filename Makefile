.PHONY: build run test lint fmt tools clean

build: bin
	go build -o ./bin/todo ./cmd/todo

run: build
	./bin/todo

test:
	go test ./...

coverage:
	go test -cover ./...

lint: ./bin/golangci-lint
	PATH=./bin:$$PATH ./bin/golangci-lint run ./...

fmt: ./bin/gofumpt ./bin/goimports
	PATH=./bin:$$PATH ./bin/gofumpt -w .
	PATH=./bin:$$PATH ./bin/goimports -w .

tools: ./bin/golangci-lint ./bin/gofumpt ./bin/goimports

bin:
	mkdir -p ./bin

./bin/golangci-lint: go.mod | bin
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
./bin/gofumpt: go.mod | bin
	GOBIN=$(CURDIR)/bin go install mvdan.cc/gofumpt@latest
./bin/goimports: go.mod | bin
	GOBIN=$(CURDIR)/bin go install golang.org/x/tools/cmd/goimports@latest

clean:
	rm -rf ./bin
