package main

import (
	"bank/kitex_gen/api"
	"context"
	"fmt"
	"time"

	//"kitex/kitex_gen/api/bankservice"
	"bank/kitex_gen/api/bank"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/circuitbreak"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/pkg/rpcinfo"

	etcd "github.com/kitex-contrib/registry-etcd"
)

func main() {
	NUM_OF_RETRIES := 5

	h := server.New(server.WithHostPorts("127.0.0.1:8887"), server.WithExitWaitTime(time.Second))

	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	h.GET("/:service/:method", func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("[Hertz] API Request Received")
		fmt.Print("[Hertz] Request: ")
		fmt.Println(string(ctx.Request.Body()))
		fmt.Println("[Hertz] Making RPC Call")

		service := ctx.Param("service")
		method := ctx.Param("method")

		fmt.Println(service)
		fmt.Println(method)

		// Parse IDL with Local Files
		p, err := generic.NewThriftFileProvider("../idl/" + service + ".thrift")
		if err != nil {
			panic(err)
		}

		g, err := generic.JSONThriftGeneric(p)
		if err != nil {
			panic(err)
		}

		var opts []client.Option

		opts = append(opts, client.WithResolver(r))

		// Circuit Breaker
		cbs := circuitbreak.NewCBSuite(GenServiceCBKeyFunc)
		opts = append(opts, client.WithCircuitBreaker(cbs))

		cli, err := genericclient.NewClient(service, g, opts...)
		if err != nil {
			panic(err)
		}

		resp, err := RpcCallWithRetry(NUM_OF_RETRIES, cli, c, method, ctx)
		if err != nil {
			panic(err)
		}

		fmt.Println("[Hertz] Response received")
		fmt.Print("[Hertz] Response: ")
		fmt.Println(resp)
		fmt.Println("Returning response to client now")

		ctx.JSON(consts.StatusOK, resp)
	})

	h.GET("/ping", func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("[Hertz] API Request Received")
		fmt.Println("[Hertz] Making RPC Call")

		client, err := bank.NewClient("BankService", client.WithResolver(r))

		// RPC client
		//client, err := bankservice.NewClient("BankService", client.WithHostPorts("0.0.0.0:8888"))
		if err != nil {
			log.Fatal(err)
		}

		req := &api.BankNameReq{Name: "my request"}
		resp, err := client.Name(context.Background(), req)
		if err != nil {
			log.Fatal(err)
		}

		ctx.JSON(consts.StatusOK, utils.H{"name": resp})
	})

	h.Spin()
}

func GenServiceCBKeyFunc(ri rpcinfo.RPCInfo) string {
	// circuitbreak.RPCInfo2Key returns "$fromServiceName/$toServiceName/$method"
	return circuitbreak.RPCInfo2Key(ri)
}

func RpcCallWithRetry(retriesLeft int, cli genericclient.Client, c context.Context, method string, ctx *app.RequestContext) (interface{}, error) {
	resp, err := cli.GenericCall(c, method, string(ctx.Request.BodyBytes()))
	if err != nil {
		if retriesLeft <= 1 {
			return nil, err
		}
		fmt.Printf("[Hertz] Retries Left: %v\n", retriesLeft)
		return RpcCallWithRetry(retriesLeft-1, cli, c, method, ctx)
	}
	return resp, nil
}
