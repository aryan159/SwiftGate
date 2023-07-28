package main

import (
	api "bank/kitex_gen/api/bank"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/limit"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	"github.com/go-redis/redis/v8"
	"github.com/hertz-contrib/obs-opentelemetry/provider"
	kitexlogrus "github.com/kitex-contrib/obs-opentelemetry/logging/logrus"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	internal_opentracing "github.com/kitex-contrib/tracer-opentracing"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

func main() {

	klog.SetLogger(kitexlogrus.NewLogger())
	klog.SetLevel(klog.LevelDebug)

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("Bank-Service"),
		provider.WithExportEndpoint("localhost:4317"),
		provider.WithInsecure(),
	)
	defer p.Shutdown(context.Background())

	var opts []server.Option

	opts = append(opts, server.WithSuite(tracing.NewServerSuite()))

	// Rate limiting
	opts = append(opts, server.WithLimit(&limit.Option{MaxConnections: 1000, MaxQPS: 125}))

	serviceName := "bank"

	//Set Up Redis to Publish config JSON
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	config, err := ReadJsonConfigFile(serviceName)

	if err == nil {
		ctx := context.Background()

		//Update allservices list
		val, err := client.Get(ctx, "allservices").Result()
		if err != nil {
			panic(err)
		}

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

		//Publish config to channel
		err = client.Publish(ctx, "services", config).Err()
		if err != nil {
			panic(err)
		}
	}

	//Setup etcd service registry
	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
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
