package main

import (
	"context"
	"fmt"
	"bank/kitex_gen/api"
	"bank/kitex_gen/api/bank"
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


	etcd "github.com/kitex-contrib/registry-etcd"

	
	"time"
	"github.com/go-redis/redis/v8"
	"github.com/hertz-contrib/cache"
	"github.com/hertz-contrib/cache/persist"

)

func main() {
	h := server.New(server.WithHostPorts("127.0.0.1:8887"))

	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	redisStore := persist.NewRedisStore(redis.NewClient(&redis.Options{
        Network: "tcp",
        Addr:    "127.0.0.1:6379",
    }))



	h.Use(cache.NewCache(
        redisStore,
        60*time.Second,
        cache.WithCacheStrategyByRequest(func(ctx context.Context, c *app.RequestContext) (bool, cache.Strategy) {
            return true, cache.Strategy{
                CacheKey: c.Request.URI().String() + string(c.Request.Body()),
            }
        }),
        cache.WithOnHitCache(func(c context.Context, ctx *app.RequestContext) {
            resp := &cache.ResponseCache{}
            redisStore.Get(c, ctx.Request.URI().String() + string(ctx.Request.Body()) , &resp)
            fmt.Println("header = " + string(resp.Header.Get("head")))
            fmt.Println("data = " + string(resp.Data))
        }),
    ))

	h.GET("/:service/:method", func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("[Hertz] API Request Received")
		fmt.Print("[Hertz] Request: ")
		fmt.Println(string(ctx.Request.Body()))
		fmt.Println("[Hertz] Making RPC Call")

		service := ctx.Param("service")
		method := ctx.Param("method")


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
		fp := retry.NewFailurePolicy()
		fp.WithMaxRetryTimes(3)

		opts = append(opts, client.WithFailureRetry(fp))

		// Circuit Breaker
		cbs := circuitbreak.NewCBSuite(GenServiceCBKeyFunc)

		opts = append(opts, client.WithCircuitBreaker(cbs))


		cli, err := genericclient.NewClient(service, g, opts...)
		if err != nil {
			panic(err)
		}

		resp, err := cli.GenericCall(c, method, string(ctx.Request.BodyBytes()))
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
		fmt.Println("[Hertz] API Request Received")
		fmt.Println("[Hertz] Making RPC Call")

		
		client, err := bank.NewClient("BankService", client.WithResolver(r))

		// RPC client
		//client, err := bankservice.NewClient("BankService", client.WithHostPorts("0.0.0.0:8888"))
		if err != nil {
			log.Fatal(err)
		}

		req := &api.BankNameReq{Name: "my request"}
		resp, err := client.Name(context.Background(), req)
		if err != nil {
			log.Fatal(err)
		}

		ctx.JSON(consts.StatusOK, utils.H{"name": resp})
	})

	h.Spin()
}

func GenServiceCBKeyFunc(ri rpcinfo.RPCInfo) string {
	// circuitbreak.RPCInfo2Key returns "$fromServiceName/$toServiceName/$method"
	return circuitbreak.RPCInfo2Key(ri)
}
