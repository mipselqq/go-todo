.PHONY: run build clean

build:
	go build -o bin/todo ./cmd/todo

run: build
	./bin/todo

clean:
	rm -rf ./bin
