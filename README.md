# SwiftGate

## Setup
Set up GOPATH correctly and clone repo to $GOPATH/src/github.com/aryan159

In the SwiftGate/kitex folder, run these 2 commands to start the rpc server
```
sh build.sh
sh output/bootstrap.sh
```
In the SwiftGate/kitex/kitex-caller folder run this command to start the client
```
go run main.go
```

You should then see "Response({Message:my request})", indicating a successful rpc call
