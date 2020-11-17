GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go
GOTEST=GO111MODULE=on go test # go race detector requires cgo

GOBUILD=$(GO) build

default: builder fuzzer

builder:
	$(GOBUILD) $(GOMOD) -o bin/tidb-fuzz-builder fuzz/cmd/builder/*.go

fuzzer:
	$(GOBUILD) $(GOMOD) -o bin/tidb-fuzz-fuzzer fuzz/cmd/fuzzer/*.go
