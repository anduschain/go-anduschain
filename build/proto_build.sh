#!/usr/bin/env bash

set -eux

# Find explicit proto roots
PROTO_ROOT_FILES=$PWD/protos/.protoroot
PROTO_ROOT_DIRS="$(dirname $PROTO_ROOT_FILES)"


# Find all proto files to be compiled, excluding any which are in a proto root or in the vendor folder
ROOTLESS_PROTO_FILES="$(find $PWD \
                             $(for dir in $PROTO_ROOT_DIRS ; do echo "-path $dir -prune -o " ; done) \
                             -name "*.proto" -exec readlink -f {} \;)"
ROOTLESS_PROTO_DIRS="$(dirname $ROOTLESS_PROTO_FILES | sort | uniq)"

for dir in $ROOTLESS_PROTO_DIRS $PROTO_ROOT_DIRS; do
echo Working on dir $dir
        # As this is a proto root, and there may be subdirectories with protos, compile the protos for each sub-directory which contains them
	for protos in $(find "$dir" -name '*.proto' -exec dirname {} \; | sort | uniq) ; do
	       protoc --proto_path="$dir" --go_out=plugins=grpc:$GOPATH/src "$protos"/*.proto
	done
done
