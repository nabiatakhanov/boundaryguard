GOEXE := $(shell go env GOEXE)
BIN := bin/boundaryguard$(GOEXE)

.PHONY: build test lint demo bench

build:
	go build -o bin/ ./cmd/boundaryguard

test:
	go test ./...

lint: build
	go run ./scripts/checkfmt
	go vet ./...
	$(BIN) -config boundaryguard.json ./...

demo:
	go run ./scripts/demo

bench:
	go run ./scripts/bench -sizes=8,32,96 -repeats=2 -out=artifacts/bench
