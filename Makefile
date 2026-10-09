.PHONY: default vet tests build clean

default: vet tests build

vet:
	go vet
	staticcheck -checks all

tests:
	go test -coverprofile="coverage.out" -test.run=. -test.v=true
	go tool cover -html="coverage.out"
	go test ./... -bench=.

build:
	go build ./cmd/fts

clean:
	@rm -f coverage.out coverage.html fts *.pprof
	@go clean
