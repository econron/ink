.PHONY: build ink view

build:
	go build cmd/main.go

ink:
	./main ink ls

view:
	./main ink view test.md

nofile:
	./main ink view
