.PHONY: run build test

run:
	go run cmd/bot/main.go

build:
	go build -o bin/github-tg-bot cmd/bot/main.go

test:
	go test -v ./...

deps:
	go mod download
