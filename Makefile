.PHONY: all rcodex rclaude rcodegen rgemini clean test

all: rcodex rclaude rcodegen rgemini

rcodex:
	go build -o rcodex ./cmd/rcodex

rclaude:
	go build -o rclaude ./cmd/rclaude

rcodegen:
	go build -o rcodegen ./cmd/rcodegen

rgemini:
	go build -o rgemini ./cmd/rgemini

clean:
	rm -f rcodex rclaude rcodegen rgemini

test:
	go test ./pkg/...
