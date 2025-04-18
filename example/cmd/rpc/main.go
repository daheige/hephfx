package main

import (
	"context"
	"fmt"
	"log"

	"github.com/daheige/hephfx/example/internal/pb"
	"github.com/daheige/hephfx/micro"
)

func main() {
	// 创建grpc微服务实例
	s := micro.NewService(
		fmt.Sprintf("0.0.0.0:%d", 50051),
		micro.WithLogger(micro.LoggerFunc(log.Printf)),
		micro.WithEnableRequestValidator(),
	)

	// 初始化greeter service
	service := &GreeterServer{}

	// 注册grpc微服务
	pb.RegisterGreeterServer(s.GRPCServer, service)

	// 运行服务
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

// GreeterServer 实现greeter微服务
type GreeterServer struct {
	// 这里必须包含这个解构体才可以，否则就是没有实现
	pb.UnimplementedGreeterServer
}

// SayHello 实现say hello方法
func (s *GreeterServer) SayHello(ctx context.Context, req *pb.HelloReq) (*pb.HelloReply, error) {
	log.Println("request id:", req.Id)
	reply := &pb.HelloReply{
		Name:    "heph-fx",
		Message: "hello world",
	}

	return reply, nil
}

// Info 实现info方法
func (s *GreeterServer) Info(ctx context.Context, req *pb.InfoReq) (*pb.InfoReply, error) {
	log.Println("request name:", req.Name)
	reply := &pb.InfoReply{
		Address: "sz",
		Message: "hello",
	}

	return reply, nil
}
