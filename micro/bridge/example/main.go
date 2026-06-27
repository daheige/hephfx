package main

import (
	"log"
	"time"

	// "github.com/daheige/hello-pb/pb"
	"google.golang.org/grpc"

	"github.com/daheige/hephfx/micro/bridge"
)

func main() {
	cfg, err := bridge.LoadConfig("./app.yaml")
	if err != nil {
		log.Fatal(err)
	}

	client, err := bridge.NewClient(cfg,
		bridge.WithServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // 全局配置
		bridge.WithServiceDialOpts("greeter-svc", grpc.WithIdleTimeout(10*time.Minute)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 调用gRPC微服务接口
	// ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// defer cancel()
	// req := &pb.HelloReq{Name: "daheige"}
	// var resp pb.HelloReply
	// if err := client.Invoke(ctx, "greeter-svc", "SayHello", req, &resp); err != nil {
	// 	log.Fatal("failed to call greeter-svc:", err)
	// }

	// fmt.Println("message:", resp.Message)
}
