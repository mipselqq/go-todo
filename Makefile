.PHONY: test run build clean

build:
	go build -o bin/todo ./cmd/todo

run: build
	./bin/todo

test: go test ./...

clean:
	rm -rf ./bin
