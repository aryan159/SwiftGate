package main

import (
	"context"
	"fmt"
	"log"
	"kitex/kitex_gen/api"
	"kitex/kitex_gen/api/bankservice"

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
