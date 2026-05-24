.PHONY: build ink render

build:
	go build cmd/main.go

ink:
	./main ink ls

render:
	./main ink render test.md

nofile:
	./main ink render