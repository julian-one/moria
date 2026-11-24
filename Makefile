BINARY := moria

.PHONY: build format lint verify test dev image

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

image:
	@docker build -t julianone/moria .
