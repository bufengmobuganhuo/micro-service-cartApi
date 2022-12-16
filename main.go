package main

import (
	"context"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	go_micro_service_cart "github.com/bufengmobuganhuo/micro-service-cart/proto/cart"
	"github.com/bufengmobuganhuo/micro-service-cartApi/handler"
	common "github.com/bufengmobuganhuo/micro-service-common"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/client"
	log "github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry"
	consul2 "github.com/micro/go-plugins/registry/consul/v2"
	"github.com/micro/go-plugins/wrapper/select/roundrobin/v2"
	opentracing2 "github.com/micro/go-plugins/wrapper/trace/opentracing/v2"
	"github.com/opentracing/opentracing-go"
	"net"
	"net/http"

	cartApi "github.com/bufengmobuganhuo/micro-service-cartApi/proto/cartApi"
)

func main() {
	// 注册中心
	consul := consul2.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{
			"127.0.0.1:8500",
		}
	})

	// 链路追踪
	t, io, err := common.NewTracer("go.micro.api.cartApi", "localhost:6831")
	if err != nil {
		log.Fatal(err)
	}
	defer io.Close()
	opentracing.SetGlobalTracer(t)

	// 熔断器
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	// 启动端口监听,会把熔断信息通过这个地址发送出去
	go func() {
		err = http.ListenAndServe(net.JoinHostPort("0.0.0.0", "9096"), hystrixStreamHandler)
		if err != nil {
			log.Error(err)
		}
	}()

	// New Service
	service := micro.NewService(
		micro.Name("go.micro.api.cartApi"),
		micro.Version("latest"),
		micro.Address("0.0.0.0:8086"),
		micro.Registry(consul),
		// api是客户端
		micro.WrapClient(opentracing2.NewClientWrapper(opentracing.GlobalTracer())),
		// 添加熔断
		micro.WrapClient(NewHystrixWrapper()),
		// 添加负载均衡
		micro.WrapClient(roundrobin.NewClientWrapper()),
	)

	// Initialise service
	service.Init()

	cartService := go_micro_service_cart.NewCartService("go.micro.service.cart", service.Client())

	// Register Handler
	cartApi.RegisterCartApiHandler(service.Server(), &handler.CartApi{CartService: cartService})

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}

type clientWrapper struct {
	client.Client
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, resp interface{}, opts ...client.CallOption) error {
	return hystrix.Do(req.Service()+"."+req.Endpoint(),
		func() error {
			fmt.Println("req.Service()" + "." + req.Endpoint())
			return c.Client.Call(ctx, req, resp, opts...)
		},
		// 出现错误时的回调
		func(err error) error {
			fmt.Println(err)
			return err
		},
	)
}

func NewHystrixWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &clientWrapper{c}
	}
}
