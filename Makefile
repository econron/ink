.PHONY: build ink view nofile test lint

build:
	go build -o ink ./cmd

test:
	go test ./...

lint:
	golangci-lint run

ink:
	./ink ls

view:
	./ink view test.md

nofile:
	./ink view

