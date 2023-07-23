package main

import (
	"context"
	"fmt"
	"io"
	"time"

	//"kitex/kitex_gen/api/bankservice"

	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/circuitbreak"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"

	hertztracer "github.com/hertz-contrib/tracer/hertz"
	etcd "github.com/kitex-contrib/registry-etcd"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

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

	h.Use(hertztracer.ServerCtx())

	h.GET("/:service/:method", func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("[Hertz] API Request Received")
		fmt.Print("[Hertz] Request: ")
		fmt.Println(string(ctx.Request.Body()))
		fmt.Println("[Hertz] Making RPC Call")

		service := ctx.Param("service")
		method := ctx.Param("method")

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

		// Circuit Breaker
		cbs := circuitbreak.NewCBSuite(GenServiceCBKeyFunc)
		opts = append(opts, client.WithCircuitBreaker(cbs))

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

	h.Spin()
}

func GenServiceCBKeyFunc(ri rpcinfo.RPCInfo) string {
	// circuitbreak.RPCInfo2Key returns "$fromServiceName/$toServiceName/$method"
	return circuitbreak.RPCInfo2Key(ri)
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
