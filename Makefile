# generate version number
version=$(shell git describe --tags --long --always --dirty|sed 's/^v//')
binfile=collectd2metrics
all:
	go build -ldflags "-X main.version=$(version)" -o $(binfile) $(binfile).go
	-@go fmt

static:
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).static $(binfile).go

arm:
	GOARCH=arm go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).arm $(binfile).go
	GOARCH=arm64 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).arm64 $(binfile).go

release:
	rm -f bin/*
	CGO_ENABLED=0 GOARCH=arm go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).arm $(binfile).go
	CGO_ENABLED=0 GOARCH=arm64 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).arm64 $(binfile).go
	CGO_ENABLED=0 GOARCH=386 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).i386 $(binfile).go
	CGO_ENABLED=0 GOARCH=amd64 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).amd64 $(binfile).go
	bash -c 'cd bin ; sha256sum * >Checksum'

version:
	@echo $(version)
