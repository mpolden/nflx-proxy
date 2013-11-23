all: fmt lint test

deps:
	go get github.com/miekg/dns

fmt:
	gofmt -tabs=false -tabwidth=4 -w=true *.go

lint:
	go vet *.go

test:
	go test
