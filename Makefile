# generate version number
version=$(shell git describe --tags --long --always --dirty|sed 's/^v//')
binfile=collectd2metrics
# CGO_EXTLDFLAGS is added for cross-compiling purpose
all:
	mkdir -p bin/
	go build -ldflags "$(CGO_EXTLDFLAGS) -X main.version=$(version)" $(binfile).go
	-@go fmt

static:
	mkdir -p bin/
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).static $(binfile).go

arm:
	mkdir -p bin/
	GOARCH=arm go build  -ldflags "$(CGO_EXTLDFLAGS) -X main.version=$(version) -extldflags \"-static\"" -o $(binfile).arm $(binfile).go
	GOARCH=arm64 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).arm64 $(binfile).go
version:
	@echo $(version)
