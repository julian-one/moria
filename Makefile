BINARY := moria

.PHONY: build format lint test dev

build:
	go build -o $(BINARY) .

format:
	golangci-lint fmt ./...

lint:
	golangci-lint run ./...

verify:
	make build && make format && make lint

test:
	go test ./...

dev:
	go run . serve
