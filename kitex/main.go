package main

import (
	api "kitex/kitex_gen/api/bankservice"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	etcd "github.com/kitex-contrib/registry-etcd"
	"log"
)

func main() {

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"}) // r should not be reused.
    if err != nil {
        log.Fatal(err)
    }

	svr := api.NewServer(new(BankServiceImpl), server.WithRegistry(r), server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "BankService"}))


	//svr := api.NewServer(new(BankServiceImpl))
	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
