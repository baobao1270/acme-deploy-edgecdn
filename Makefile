BINARY  := acme-deploy-edgecdn
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
DIST    := dist

RELEASE_TARGETS := \
	$(DIST)/$(BINARY)-darwin-arm64 \
	$(DIST)/$(BINARY)-linux-arm64 \
	$(DIST)/$(BINARY)-linux-386 \
	$(DIST)/$(BINARY)-linux-amd64

.PHONY: build release clean

build:
	go build -o $(DIST)/$(BINARY) .

release: $(RELEASE_TARGETS)

$(DIST)/$(BINARY)-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags '$(LDFLAGS)' -o $@ .

$(DIST)/$(BINARY)-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags '$(LDFLAGS)' -o $@ .

$(DIST)/$(BINARY)-linux-386:
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags '$(LDFLAGS)' -o $@ .

$(DIST)/$(BINARY)-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -o $@ .

clean:
	rm -rvf $(DIST) $(BINARY)
