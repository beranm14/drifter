all: build

build: build_linux

build_linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o drifter ./*.go
