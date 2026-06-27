package gclient

import (
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/daheige/hephfx/gutils"
)

var (
	// the gRPC client makes a connection call to the connection management.
	gRPCConnections = map[string]*grpc.ClientConn{}
	mu              sync.Mutex
)

// RegisterGRPCConnections gRPC client connection manage
// If there are multiple gRPC client calls in a project, it is recommended to manage them uniformly.
// Before the main function of the program exits, all of them can be closed by
// calling defer gclient.CloseGRPCConnections() method,
// thereby releasing the gRPC connections.
//
// Of course, you can also close the current connection through gclient.Close(target)
func RegisterGRPCConnections(target string, conn *grpc.ClientConn) {
	mu.Lock()
	defer mu.Unlock()

	key := gutils.Md5("client-" + target)
	_, ok := gRPCConnections[key]
	if ok {
		log.Printf("gRPC connection target:%s already exists", target)
		return
	}

	gRPCConnections[key] = conn
}

// Close for the specified target gRPC client connection closing
func Close(target string) error {
	mu.Lock()
	defer mu.Unlock()

	key := gutils.Md5("client-" + target)
	conn, ok := gRPCConnections[key]
	if !ok {
		return fmt.Errorf("gRPC connection target:%s not found", target)
	}

	err := conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close gRPC connection err:%w", err)
	}

	delete(gRPCConnections, key)
	return nil
}

// CloseGRPCConnections When the program exits, after the gRPC client call is completed,
// it is generally necessary to close it in the main method of the main.go file.
// It needs to be called use defer gclient.CloseGRPCConnections() method.
func CloseGRPCConnections() {
	mu.Lock()
	defer mu.Unlock()

	for k := range gRPCConnections {
		err := gRPCConnections[k].Close()
		if err != nil {
			log.Printf("gRPC connection key:%s close error:%v\n", k, err)
		}

		// 删除连接
		delete(gRPCConnections, k)
	}
}

// InitGRPCClient creates a gRPC connection and generates any pb.XXXClient through a factory,
// T represents the client interface type, such as pb.GreeterClient, pb.OrderClient, etc.
// target represents the endpoint connection address, which can be either a host:port or a k8s named service address.
func InitGRPCClient[T any](target string, factory func(grpc.ClientConnInterface) T,
	opts ...grpc.DialOption) (T, error) {
	var zero T
	// 如果使用k8s命名服务以及headless方式访问，实现客户端负载均衡
	// 关键配置：默认启用round_robin负载均衡策略
	options := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithIdleTimeout(30 * time.Minute), // 连接生命周期
		grpc.WithMaxCallAttempts(3),            // 最大重试次数
	}
	if len(opts) > 0 {
		options = append(options, opts...)
	}
	conn, err := grpc.NewClient(target, options...)
	if err != nil {
		return zero, err
	}

	// 注册连接管理
	RegisterGRPCConnections(target, conn)

	return factory(conn), nil
}
