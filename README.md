# SwiftGate

## General Info

The project is an API Gateway powered by [CloudWeGo](https://github.com/cloudwego). It consists of 2 part, the [Hertz](https://github.com/cloudwego/hertz) server that handles incoming HTTP requests and forwards it to the [Kitex](https://github.com/cloudwego/kitex) server that handles the RPC services.

## Technologies

Make sure to have GO installed on your device. Please follow the instructions [here](https://go.dev/doc/install) to install GO.

This program contains indirect calls to [NetPoll](https://github.com/cloudwego/netpoll) which is not compatible with Windows, to ensure the program can run, it is recommended to use Linux or MacOS. Windows users can explore [WSL](https://learn.microsoft.com/en-us/windows/wsl/install).

Set up GOPATH correctly if you are having issues with modules please enable GO111MODULE with the command below

```
go env -w GO111MODULE=on
```

Our API Gateway uses etcd as a service registery, for setup and install of the etcd service, please follow the guide [here](https://github.com/etcd-io/etcd/releases)


To set up redis for caching, follow [this](https://redis.io/docs/getting-started/installation/) guide to install.


## Setup

1. Download and Install Docker
2. Run these scripts

```
sh scripts/jaeger_run.sh
sh scripts/hertz_server.sh
sh scripts/bank_service.sh
```

Once both servers starts up, you can start sending HTTP GET requests to your localhost at port 8887 with tools such as Postman.

```
http://127.0.0.1:8887
```

Access the bank name service with `http://127.0.0.1:8887/bank/name` with the body as json in the format below:

```
{
  "Name":"INSERT_NAME"
}
```

Then, you can monitor the generated traces at http://localhost:16686
