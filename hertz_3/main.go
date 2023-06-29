package main

import (
	"context"
	"fmt"
	"kitex/kitex_gen/api"
	"kitex/kitex_gen/api/bankservice"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/generic"
)

func main() {
	h := server.New(server.WithHostPorts("127.0.0.1:8887"))

	h.GET("/bank/name", func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("[Hertz] API Request Received")
		fmt.Println("[Hertz] Making RPC Call")

		// Parse IDL with Local Files
		// YOUR_IDL_PATH thrift file path, eg:./idl/example.thrift
		p, err := generic.NewThriftFileProvider("../idl_2/bank_api.thrift")
		if err != nil {
			panic(err)
		}
		g, err := generic.JSONThriftGeneric(p)
		if err != nil {
			panic(err)
		}
		cli, err := genericclient.NewClient("BankService", g, client.WithHostPorts("0.0.0.0:8888"))
		if err != nil {
			panic(err)
		}
		// 'ExampleMethod' method name must be passed as param
		resp, err := cli.GenericCall(c, "GetNameMethod", "{\"name\": \"random\"}")
		if err != nil {
			panic(err)
		}
		// resp is a JSON string

		ctx.JSON(consts.StatusOK, utils.H{"name": resp})
	})

	h.GET("/ping", func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("[Hertz] API Request Received")
		fmt.Println("[Hertz] Making RPC Call")

		// RPC client
		client, err := bankservice.NewClient("BankService", client.WithHostPorts("0.0.0.0:8888"))
		if err != nil {
			log.Fatal(err)
		}

		req := &api.BankNameReq{Name: "my request"}
		resp, err := client.GetNameMethod(context.Background(), req)
		if err != nil {
			log.Fatal(err)
		}

		ctx.JSON(consts.StatusOK, utils.H{"name": resp})
	})

	h.Spin()
}
