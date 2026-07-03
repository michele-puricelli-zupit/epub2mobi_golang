.PHONY: build build-gui test run run-gui

build:
	go build -o bin/epub2mobi ./cmd/epub2mobi

build-gui:
	go build -o bin/epub2mobi-gui ./cmd/epub2mobi-gui

test:
	go test ./...

run:
	go run ./cmd/epub2mobi -in ./ebooks -out ./converted

run-gui:
	go run ./cmd/epub2mobi-gui
