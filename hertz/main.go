package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/circuitbreak"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/go-redis/redis/v8"
	"github.com/hertz-contrib/cache"
	"github.com/hertz-contrib/cache/persist"
	"github.com/hertz-contrib/keyauth"
	hertzlogrus "github.com/hertz-contrib/obs-opentelemetry/logging/logrus"
	"github.com/hertz-contrib/obs-opentelemetry/provider"
	hertztracing "github.com/hertz-contrib/obs-opentelemetry/tracing"
	kitextracing "github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

// Global vairable to store all the current active configs
var servicesConfig map[string]map[string]Config

func main() {

	//Set up logger
	hlog.SetLogger(hertzlogrus.NewLogger())
	hlog.SetLevel(hlog.LevelDebug)

	hlog.Debug("Hertz::main\n")

	//Set up OpenTelemetry
	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("Hertz-Server"),
		// Support setting ExportEndpoint via environment variables: OTEL_EXPORTER_OTLP_ENDPOINT
		provider.WithExportEndpoint("localhost:4317"),
		provider.WithInsecure(),
	)
	defer p.Shutdown(context.Background())

	tracer, cfg := hertztracing.NewServerTracer()

	//Set up Hertz server
	h := server.New(
		server.WithHostPorts("127.0.0.1:8887"),
		server.WithExitWaitTime(time.Second),
		tracer)

	h.Use(hertztracing.ServerMiddleware(cfg))

	//Set up etcd registry
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	//Setup Redis for caching
	redisStore := persist.NewRedisStore(redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "127.0.0.1:6379",
	}))

	//Load all the config of the current running services
	LoadAllServices()

	//Start the listener for the published channels
	go startRedisSubscription()

	v1 := h.Group("/:service/:method")

	//Setup Keyauth
	v1.Use(keyauth.New(
		keyauth.WithFilter(func(c context.Context, ctx *app.RequestContext) bool {
			return !servicesConfig[string(ctx.Param("service"))][string(ctx.Param("method"))].Auth.EnableAuth
		}),
	))

	//Setup Caching
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

	v1.GET("/.", func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("[Hertz] API Request Received")
		fmt.Print("[Hertz] Request: ")
		fmt.Println(string(ctx.Request.Body()))

		service := ctx.Param("service")
		method := ctx.Param("method")

		value, _ := ctx.Get("token")

		//Process the auth token
		if servicesConfig[service][method].Auth.EnableAuth {
			if value != servicesConfig[service][method].Auth.Token {
				ctx.SetStatusCode(consts.StatusUnauthorized)
				return
			}
		}

		// Parse IDL with Local Files
		p, err := generic.NewThriftFileProvider("../idl/" + service + ".thrift")
		if err != nil {
			fmt.Print(err.Error())
			ctx.SetStatusCode(consts.StatusServiceUnavailable)
			return
		}

		g, err := generic.JSONThriftGeneric(p)
		if err != nil {
			fmt.Print(err.Error())
			ctx.SetStatusCode(consts.StatusServiceUnavailable)
			return
		}

		var opts []client.Option

		// tracer
		opts = append(opts, client.WithSuite(kitextracing.NewClientSuite()))

		// etcd
		opts = append(opts, client.WithResolver(r))

		// Circuit Breaker
		if servicesConfig[service][method].CircuitBreaker.EnableCircuitBreaker {
			cbs := circuitbreak.NewCBSuite(GenServiceCBKeyFunc)
			opts = append(opts, client.WithCircuitBreaker(cbs))
		}

		cli, err := genericclient.NewClient(service, g, opts...)
		if err != nil {
			fmt.Print(err.Error())
			ctx.SetStatusCode(consts.StatusServiceUnavailable)
			return
		}

		fmt.Println("[Hertz] Making RPC Call")

		//Retry
		numOfTries := 1
		if servicesConfig[service][method].Retry.EnableRetry {
			numOfTries = servicesConfig[service][method].Retry.MaxTimes
		}

		resp, err := RpcCallWithRetry(numOfTries, cli, c, method, ctx)
		if err != nil {
			fmt.Printf("[RPC Error] %v", err.Error())
			if strings.Contains(err.Error(), "dependency error") {
				ctx.SetStatusCode(consts.StatusFailedDependency)
			} else {
				ctx.SetStatusCode(consts.StatusTooManyRequests)
			}
			return
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
	return circuitbreak.RPCInfo2Key(ri)
}

func LoadAllServices() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx2 := context.Background()
	val, err := client.Get(ctx2, "allservices").Result()
	if err != nil {
		panic(err)
	}

	if val == "" {
		return
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

func InitTracer(serviceName string) (opentracing.Tracer, io.Closer) {
	cfg, _ := jaegercfg.FromEnv()
	cfg.ServiceName = serviceName
	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
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
