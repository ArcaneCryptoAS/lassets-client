## Lightning Assets Client
This project is the client side of a naive implementation of [lightning
 assets](http://research.paradigm.xyz/RainbowNetwork.pdf).

#### What are lightning assets?
With this project, you can circumvent the problem of volatility in Bitcoin, and "peg" your lightning balance to
a stable(or unstable) currency of your choice. This can be your local currency, dollar, euro, ethereum etc.
It is non-custodial, meaning noone can take money from you, yay :tada:

#### How does it work
As a client, you open "contracts" with the server. The client and server will continuously rebalance the
contract, by sending sats client --> server if the price has dropped, or server --> client if the price
has increased. The goal is to ensure the balance of the contract is always X [ASSET]. The asset of the contract
can be any asset both the client and server support. To determine the price of the contract, the server and
client have to agree on an oracle. As of today, this common price feed is [bitmex](https://bitmex.com).

The client comes configured out of the box to connect to a server we are running.

### Installing  
First download the project
```
go get -u github.com/ArcaneCryptoAS/lassets-client
```

cd to the project:
```
cd $GOPATH/github.com/ArcaneCryptoAS/lassets-client
```

To install the project, run in the project root directory
```
go install ./...
```
This will install two new commands in your $GOBIN
```
laccli
lacd
```

If you already have a lnd-node you want to connect to, run `lacd --network=testnet` and everything should connect.

Try to open a contract!
```shell script
laccli opencontract --amount=5 --asset=USD # opens contract for 5 usd
```

To start the client daemon on regtest, you first need a [Lightning Assets Server](https://github.com/ArcaneCryptoAS/lassets-server) running, then run:
```
./lacd
```

### Required dependencies

### lnd
The project requires a lnd node running on your machine, regtest, testnet and
 mainnet is supported. Check out the official repo for installation
  instructions: https://github.com/lightningnetwork/lnd


### Optional dependencies
Only required if you want to make changes to the .proto files
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

### Contributions 
Contributions are very welcome, just go ahead and open issues/pull requests.
