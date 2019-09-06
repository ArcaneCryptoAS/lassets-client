#!/bin/sh

set -e

# Generate the gRPC bindings for all proto files.
for file in *.proto
do
	protoc -I/usr/local/include -I. \
	       -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	       -I$GOPATH/src/gitlab.com/arcanecrypto \
	       --go_out=plugins=grpc,paths=source_relative:. \
	       --grpc-gateway_out=logtostderr=true,paths=source_relative:. \
		${file}

done