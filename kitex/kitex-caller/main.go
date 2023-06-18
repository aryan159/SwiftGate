package main

import (
	"context"
	"log"
	"rpc/kitex_gen/api"
	"rpc/kitex_gen/api/echo"
	"time"

	"github.com/cloudwego/kitex/client"
)

func main() {
	client, err := echo.NewClient("hello", client.WithHostPorts("0.0.0.0:8888"))
	if err != nil {
		log.Fatal(err)
	}
	for {
		req := &api.Request{Message: "my request"}
		resp, err := client.Echo(context.Background(), req)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(resp)
		time.Sleep(time.Second)
	}
}
