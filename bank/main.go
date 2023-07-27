package main

import (
	api "bank/kitex_gen/api/bank"
	"fmt"
	"log"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	etcd "github.com/kitex-contrib/registry-etcd"

	"io/ioutil"
	"os"

	"context"

	"github.com/go-redis/redis/v8"
)

func main() {

	serviceName := "bank"

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	config, err := ReadJsonConfigFile(serviceName)

	if err == nil {
		ctx := context.Background()
		err = client.Set(ctx, serviceName, config, 0).Err()
		if err != nil {
			panic(err)
		}
	}

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"}) // r should not be reused.
	if err != nil {
		log.Fatal(err)
	}

	svr := api.NewServer(new(BankImpl), server.WithRegistry(r), server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: serviceName}))

	err = svr.Run()

	RemoveServiceFromRedis(serviceName)

	if err != nil {
		log.Println(err.Error())
	}

}

func ReadJsonConfigFile(service string) (string, error) {

	jsonFile, err := os.Open("../config/" + service + "_config.json")

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully Opened " + service + "_config.json")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	return string(byteValue), nil

}

func RemoveServiceFromRedis(service string) {

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	err := client.Del(ctx, service).Err()

	if err != nil {
		panic(err)
	}

}
