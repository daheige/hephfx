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
	log.Println("address: ", address)

	// Set up a connection to the server.
	clientConn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	defer clientConn.Close()

	client := pb.NewGreeterClient(clientConn)

	// Contact the server and print out its response.
	res, err := client.SayHello(context.Background(), &pb.HelloReq{
		Id: 1,
	})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("name:%s,message:%s", res.Name, res.Message)

	res2, err := client.Info(context.Background(), &pb.InfoReq{
		Name: "daheige",
	})

	log.Println(res2, err)
}
