package main

import (
	api "bank/kitex_gen/api/bank"
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


	svr := api.NewServer(new(BankImpl), server.WithRegistry(r), server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "bank"}))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
