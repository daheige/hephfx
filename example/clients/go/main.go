package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/daheige/hephfx/example/clients/go/pb"
)

func main() {
	address := "localhost:50051"
	// 或者使用k8s命名服务地址: hello.svc.local:50051
	// 使用k8s命名服务+dns解析方式连接，格式:dns:///your-service.namespace.svc.cluster.local:50051
	// address := "dns:///hello.test.svc.cluster.local:50051"
	log.Println("address: ", address)

	// Set up a connection to the server.
	clientConn, err := grpc.NewClient(
		address,
		// grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	defer clientConn.Close()

	client := pb.NewGreeterClient(clientConn)

	// Contact the server and print out its response.
	res, err := client.SayHello(context.Background(), &pb.HelloReq{
		Name: "daheige",
	})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("res message:%s", res.Message)
}
