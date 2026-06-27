package main

import (
	"context"
	"log"

	"google.golang.org/grpc"

	"github.com/daheige/hephfx/example/clients/go/pb"
	"github.com/daheige/hephfx/micro/gclient"
)

func main() {
	address := "localhost:50051"
	// 或者使用k8s命名服务地址: hello.svc.local:50051
	// 使用k8s命名服务+dns解析方式连接，格式:dns:///your-service.namespace.svc.cluster.local:50051
	// address := "dns:///hello.test.svc.cluster.local:50051"
	log.Println("address:", address)

	// create gRPC client
	client, err := gclient.InitGRPCClient(address, pb.NewGreeterClient, grpc.WithMaxCallAttempts(3))
	if err != nil {
		log.Fatalf("failed to create gRPC client: %v", err)
	}
	defer gclient.Close(address)

	// Contact the server and print out its response.
	res, err := client.SayHello(context.Background(), &pb.HelloReq{
		Name: "daheige",
	})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("res message:%s", res.Message)
}
