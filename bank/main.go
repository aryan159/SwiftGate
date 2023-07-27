package main

import (
	api "bank/kitex_gen/api/bank"
	"fmt"
	"io"
	"log"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	etcd "github.com/kitex-contrib/registry-etcd"

	"io/ioutil"
	"os"

	"context"

	"github.com/go-redis/redis/v8"

	internal_opentracing "github.com/kitex-contrib/tracer-opentracing"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"strings"
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

		val, err := client.Get(ctx, "allservices").Result()

		if !strings.Contains(val, serviceName) {
			val += serviceName + ";"
			err = client.Set(ctx, "allservices", val, 0).Err()
			if err != nil {
				panic(err)
			}
		}

		err = client.Set(ctx, serviceName, config, 0).Err()
		if err != nil {
			panic(err)
		}

		err = client.Publish(ctx, "services", config).Err()
		if err != nil {
			panic(err)
		}
	}
	var opts []server.Option

	tracerSuite, closer := InitJaeger("kitex-server")
	defer closer.Close()
	opts = append(opts, server.WithSuite(tracerSuite))

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"}) // r should not be reused.
	if err != nil {
		log.Fatal(err)
	}
	opts = append(opts, server.WithRegistry(r))

	opts = append(opts, server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: serviceName}))

	svr := api.NewServer(new(BankImpl), opts...)

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

	val, err := client.Get(ctx, "allservices").Result()

	if strings.Contains(val, service) {
		val = strings.Replace(val, service+";", "", 1)
		err = client.Set(ctx, "allservices", val, 0).Err()
		if err != nil {
			panic(err)
		}
	}

}

func InitJaeger(service string) (server.Suite, io.Closer) {
	cfg, _ := jaegercfg.FromEnv()
	cfg.ServiceName = service
	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	opentracing.InitGlobalTracer(tracer)
	return internal_opentracing.NewDefaultServerSuite(), closer
}
