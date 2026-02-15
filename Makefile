VERSION := $(shell cat VERSION | tr -d '[:space:]')
LDFLAGS := -X rcodegen/pkg/runner.Version=$(VERSION)
BINDIR  := bin

.PHONY: all rclaude rcodex rgemini rcodegen clean test

all: rclaude rcodex rgemini rcodegen

rclaude:
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/rclaude ./cmd/rclaude

rcodex:
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/rcodex ./cmd/rcodex

rgemini:
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/rgemini ./cmd/rgemini

rcodegen:
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/rcodegen ./cmd/rcodegen

clean:
	rm -f $(BINDIR)/rclaude $(BINDIR)/rcodex $(BINDIR)/rgemini $(BINDIR)/rcodegen

test:
	go test ./pkg/...
