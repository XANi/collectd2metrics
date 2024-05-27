# generate version number
version=$(shell git describe --tags --long --always --dirty|sed 's/^v//')
binfile=collectd2metrics
# CGO_EXTLDFLAGS is added for cross-compiling purpose
all:
	go build -ldflags "$(CGO_EXTLDFLAGS) -X main.version=$(version)" $(binfile).go
	-@go fmt

static:
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).static $(binfile).go

arm:
	GOARCH=arm go build  -ldflags "$(CGO_EXTLDFLAGS) -X main.version=$(version) -extldflags \"-static\"" -o $(binfile).arm $(binfile).go
	GOARCH=arm64 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).arm64 $(binfile).go
version:
	@echo $(version)
