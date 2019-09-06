### Required dependencies

#### grpc-gateway
Installation instructions copied from [official repo](https://github.com/grpc).

The grpc-gateway requires a local installation of the Google protocol buffers
 compiler protoc v3.0.0 or above. To check if you already have this installed
 , run `protoc --version`. If you do not, please install this via your local
  package manager or by downloading one of the releases from the official repository:
  
https://github.com/protocolbuffers/protobuf/releases

Then use go get -u to download the following packages:

```bash
go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
go get -u github.com/golang/protobuf/protoc-gen-go
```
This will place three binaries in your $GOBIN;
```text
protoc-gen-grpc-gateway
protoc-gen-swagger
protoc-gen-go
```

Make sure that your $GOBIN is in your $PATH.

