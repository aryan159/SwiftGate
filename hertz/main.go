package main

import (
	"context"
	"fmt"
	"log"
	"rpc/kitex_gen/api"
	"rpc/kitex_gen/api/echo"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
)

func main() {
	h := server.New(server.WithHostPorts("127.0.0.1:8887"))

	h.GET("/ping", func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("[Hertz] API Request Received")
		fmt.Println("[Hertz] Making RPC Call")

		// RPC client
		client, err := echo.NewClient("hello", client.WithHostPorts("0.0.0.0:8886"))
		if err != nil {
			log.Fatal(err)
		}

		req := &api.Request{Message: "my request"}
		resp, err := client.Echo(context.Background(), req)
		if err != nil {
			log.Fatal(err)
		}

		ctx.JSON(consts.StatusOK, utils.H{"message": resp})
	})

	h.Spin()
}
