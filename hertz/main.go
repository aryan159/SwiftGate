package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/circuitbreak"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/pkg/retry"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"

	etcd "github.com/kitex-contrib/registry-etcd"

	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hertz-contrib/cache"
	"github.com/hertz-contrib/cache/persist"

	"encoding/json"

	"github.com/hertz-contrib/keyauth"
	hertztracer "github.com/hertz-contrib/tracer/hertz"
	etcd "github.com/kitex-contrib/registry-etcd"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

var servicesConfig map[string]map[string]Config

func main() {
	NUM_OF_RETRIES := 5

	hertzTracer, hertzTracerCloser := InitTracer("hertz-server")
	defer hertzTracerCloser.Close()

	h := server.New(server.WithHostPorts("127.0.0.1:8887"), server.WithExitWaitTime(time.Second),
		server.WithTracer(hertztracer.NewTracer(hertzTracer, func(c *app.RequestContext) string {
			return "test.hertz.server" + "::" + c.FullPath()
		})))

	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	redisStore := persist.NewRedisStore(redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "127.0.0.1:6379",
	}))

	LoadAllServices()
	go startRedisSubscription()

	v1 := h.Group("/:service/:method")

	v1.Use(keyauth.New(
		keyauth.WithFilter(func(c context.Context, ctx *app.RequestContext) bool {
			return !servicesConfig[string(ctx.Param("service"))][string(ctx.Param("method"))].Auth.EnableAuth
		}),
	))

	v1.Use(cache.NewCache(
		redisStore,
		60*time.Second,
		cache.WithCacheStrategyByRequest(func(ctx context.Context, c *app.RequestContext) (bool, cache.Strategy) {

			service := string(c.Param("service"))
			method := string(c.Param("method"))

			enableCaching := servicesConfig[service][method].Cache.EnableCache

			cacheKey := ""

			if servicesConfig[service][method].Cache.WithURI {
				cacheKey += c.Request.URI().String()
			}
			if servicesConfig[service][method].Cache.WithHeader {
				cacheKey += c.Request.Header.String()
			}
			if servicesConfig[service][method].Cache.WithBody {
				cacheKey += string(c.Request.Body())
			}

			return enableCaching, cache.Strategy{
				CacheKey: cacheKey,
			}
		}),
		cache.WithOnHitCache(func(c context.Context, ctx *app.RequestContext) {
			fmt.Println("[Hertz] CACHE HIT")
			fmt.Println("[Hertz] Returning Cached Response")
		}),
	))

	v1.Use(hertztracer.ServerCtx())

	v1.GET("/.", func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("[Hertz] API Request Received")
		fmt.Print("[Hertz] Request: ")
		fmt.Println(string(ctx.Request.Body()))

		service := ctx.Param("service")
		method := ctx.Param("method")

		value, _ := ctx.Get("token")

		if servicesConfig[service][method].Auth.EnableAuth {
			if value != servicesConfig[service][method].Auth.Token {
				ctx.SetStatusCode(consts.StatusUnauthorized)
				return
			}
		}
		fmt.Printf("Service: %v, Method, %v\n", service, method)

		// Parse IDL with Local Files
		p, err := generic.NewThriftFileProvider("../idl/" + service + ".thrift")
		if err != nil {
			panic(err)
		}

		g, err := generic.JSONThriftGeneric(p)
		if err != nil {
			panic(err)
		}

		var opts []client.Option

		opts = append(opts, client.WithResolver(r))

		// Retry

		if servicesConfig[service][method].Retry.EnableRetry {
			fp := retry.NewFailurePolicy()
			fp.WithMaxRetryTimes(servicesConfig[service][method].Retry.MaxTimes)

			opts = append(opts, client.WithFailureRetry(fp))
		}

		// Circuit Breaker
		if servicesConfig[service][method].CircuitBreaker.EnableCircuitBreaker {
			cbs := circuitbreak.NewCBSuite(GenServiceCBKeyFunc)
			opts = append(opts, client.WithCircuitBreaker(cbs))
		}

		// Tracing
		// kitexTracer, kitexTracerCloser := InitTracer("kitex-client")
		// defer kitexTracerCloser.Close()
		// opts = append(opts, client.WithSuite(kopentracing.NewClientSuite(kitexTracer, func(c context.Context) string {
		// 	endpoint := rpcinfo.GetRPCInfo(c).From()
		// 	return endpoint.ServiceName() + "::" + endpoint.Method()
		// })))

		cli, err := genericclient.NewClient(service, g, opts...)
		if err != nil {
			panic(err)
		}

		fmt.Println("[Hertz] Making RPC Call")

		resp, err := RpcCallWithRetry(NUM_OF_RETRIES, cli, c, method, ctx)
		if err != nil {
			panic(err)
		}

		fmt.Println("[Hertz] Response received")
		fmt.Print("[Hertz] Response: ")
		fmt.Println(resp)
		fmt.Println("Returning response to client now")

		ctx.JSON(consts.StatusOK, resp)
	})

	h.GET("/ping", func(c context.Context, ctx *app.RequestContext) {
		ctx.JSON(consts.StatusOK, utils.H{"message": "pong"})
	})

	h.Spin()

}

func GenServiceCBKeyFunc(ri rpcinfo.RPCInfo) string {
	// circuitbreak.RPCInfo2Key returns "$fromServiceName/$toServiceName/$method"
	return circuitbreak.RPCInfo2Key(ri)
}

func LoadAllServices() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	val, err := client.Get(ctx, "allservices").Result()
	if err != nil {
		panic(err)
	}

	for _, element := range strings.Split(val, ";") {
		if element != "" {
			ReadConfigFromRedis(element)
		}
	}

}

func ReadConfigFromRedis(service string) {

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	val, err := client.Get(ctx, service).Result()
	if err != nil {
		panic(err)
	}

	var result map[string]map[string]Config

	json.Unmarshal([]byte(val), &result)

	if servicesConfig == nil {
		servicesConfig = make(map[string]map[string]Config)
	}

	servicesConfig[service] = result[service]

}

func startRedisSubscription() {

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	pubsub := client.Subscribe(ctx, "services")

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			panic(err)
		}

		var result map[string]map[string]Config

		json.Unmarshal([]byte(msg.Payload), &result)

		if servicesConfig == nil {
			servicesConfig = make(map[string]map[string]Config)
		}

		keys := make([]string, 0, len(result))
		for k := range result {
			keys = append(keys, k)
		}

		servicesConfig[keys[0]] = result[keys[0]]
	}

}

func RpcCallWithRetry(retriesLeft int, cli genericclient.Client, c context.Context, method string, ctx *app.RequestContext) (interface{}, error) {
	resp, err := cli.GenericCall(c, method, string(ctx.Request.BodyBytes()))
	if err != nil {
		if retriesLeft <= 1 {
			return nil, err
		}
		fmt.Printf("[Hertz] Retries Left: %v\n", retriesLeft)
		return RpcCallWithRetry(retriesLeft-1, cli, c, method, ctx)
	}
	return resp, nil
}

// InitTracer Initialize and create tracer
func InitTracer(serviceName string) (opentracing.Tracer, io.Closer) {
	cfg, _ := jaegercfg.FromEnv()
	cfg.ServiceName = serviceName
	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	// opentracing.InitGlobalTracer(tracer)
	return tracer, closer
}
