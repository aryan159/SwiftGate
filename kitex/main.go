package main

import (
	"log"
	"net"
	api "rpc/kitex_gen/api/echo"

	"github.com/cloudwego/kitex/server"
)

func main() {
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8886")
	svr := api.NewServer(new(EchoImpl), server.WithServiceAddr(addr))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
