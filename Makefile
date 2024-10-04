# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: godaon evm all test clean
.PHONY: fairnode loadtest bootnode-linux-amd64 proto

GOBIN = $(shell pwd)/build/bin
GO ?= latest

godaon:
	build/env.sh go run build/ci.go install ./cmd/godaon
	@echo "Done building godaon."
	@echo "Run \"$(GOBIN)/godaon\" to launch godaon."

fairnode:
	build/env.sh go run build/ci.go install ./cmd/fairnode
	@echo "Done building fairnode."
	@echo "Run \"$(GOBIN)/fairnode\" to launch fairnode."

loadtest:
	build/env.sh go run build/ci.go install ./cmd/loadtest
	@echo "Done building loadtest."
	@echo "Run \"$(GOBIN)/loadtest\" to launch loadtest."

all:
	build/env.sh go run build/ci.go install
	@echo "Done building all."

test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

proto:
	build/proto_build.sh
	@echo "Done building proto file."

clean:
	./build/clean_go_build_cache.sh
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'
