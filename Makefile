.PHONY: all rcodex rclaude clean test

all: rcodex rclaude

rcodex:
	go build -o rcodex ./cmd/rcodex

rclaude:
	go build -o rclaude ./cmd/rclaude

clean:
	rm -f rcodex rclaude

test:
	go test ./pkg/...
