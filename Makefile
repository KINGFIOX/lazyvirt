BINARY  := bin/lazyvirt
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build run test integration vet fmt clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

run: build
	./$(BINARY)

test:
	go test ./...

integration:
	go test -tags integration ./internal/virsh/ -run Integration -v

vet:
	go vet -tags integration ./...

fmt:
	gofmt -l -w .

clean:
	rm -f $(BINARY)
