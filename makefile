all: server

server:
	go mod tidy
	go build -o bin/im-demo ./
