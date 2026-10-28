.PHONY: all build frontend backend test clean run

all: build

frontend:
	cd frontend && pnpm install && pnpm build

backend:
	go build -ldflags="-s -w" -o bin/server ./cmd/server

build: frontend backend

test:
	go test -v ./...

run: build
	./bin/server

clean:
	rm -rf bin/ web/dist/* data/*.db*
