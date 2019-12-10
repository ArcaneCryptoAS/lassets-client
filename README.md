## Lightning Assets Client
This project is the client side of a naive implementation of [lightning
 assets](http://research.paradigm.xyz/RainbowNetwork.pdf).

The purpose of this client is to create synthetic assets on the lightning
 network, by opening "contracts" with a server. The client and server will
  continuously rebalance the contract, thereby making sure the satoshi balance
   between them is always X [ASSET]. The asset of the contract
    can be any asset both the client and server support. To determine the
     price of the contract, the server and client have to agree on an oracle.

The server side will be open-sourced soon, and we will also host a server if
 you just want to run the client part.
 
### Installing  
To install the project, run in the project root directory
```
go install ./...
```
This will install two new commands in your $GOBIN
```
laccli
lacd
```

Output from lacd:
```
➜ lacd -h
NAME:
   ladclient - client daemon for Lightning Assets

USAGE:
   lacd [global options] command [command options] [arguments...]
   
VERSION:
   0.0.1-alpha
   
COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --port value                 port to run lightning asset grpc daemon on (default: 10456)
   --restport value             port to run lightning asset rest server (default: 8081)
   --laddir value               the location of lad dir (default: "/Users/USER
/.lac")
   --network value              which bitcoin network to run on, regtest | testnet | mainnet (default: "regtest")
   --rebalancefrequency value   how often to rebalance channels (default: 0)
   --netaddress value           our ip-address, that other hosts can reach us at (default: "localhost:10456")
   --priceserver_address value  host:port the price server is running on (default: "http://127.0.0.1:3001")
   --serveraddress value        the host:port the asset server is running on (default: "localhost:10455")
   --insecureserver             whether the connection to the server should use TLS or not
   --lnddir value               the full path to lnd directory (default: "/Users/USER/.lnd")
   --lndrpchost value           host:port of lnd daemon (default: "localhost:10011")
   --help, -h                   show help
   --version, -v                print the version
```

Output from laccli:
```
➜ laccli -h
NAME:
   laccli - control plane for your Lightning Assets Client Daemon (lac)

USAGE:
   laccli [global options] command [command options] [arguments...]
   
VERSION:
   0.0.1-alpha
   
COMMANDS:
     help, h  Shows a list of commands or help for one command

   Contracts:
     opencontract   Open a new contract with another lightning asset server
     closecontract  Close a contract
     listcontracts  list all open contracts

GLOBAL OPTIONS:
   --rpcport value  port to listen for grpc connections on (default: 10456)
   --help, -h       show help
   --version, -v    print the version
```

Because the server side of the project is not open-sourced yet, it is not yet
 possible to run the project. The server will be open-sourced in a few days.
 
### Contributions 
Contributions are very welcome, just go ahead and open issues/pull requests.

### Required dependencies

### lnd
The project requires a lnd node running on your machine, regtest, testnet and
 mainnet is supported. Check out the official repo for installation
  instructions: https://github.com/lightningnetwork/lnd

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

