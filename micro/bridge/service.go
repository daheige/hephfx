package bridge

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

// Service 封装单个下游 gRPC 服务的连接与调用。
type Service struct {
	cfg  *ServiceConfig
	conn *grpc.ClientConn
}

func newService(cfg *ServiceConfig, opts ClientOptions) (*Service, error) {
	dialOpts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(opts.serviceConfig), // 客户端负载均衡策略
		grpc.WithIdleTimeout(opts.idleTimeout),            // 连接生命周期
		grpc.WithMaxCallAttempts(opts.maxCallAttempts),    // 最大重试次数
	}, opts.options...)

	// 合并服务的 grpc.DialOption
	if extra, ok := opts.serviceDialOpts[cfg.Name]; ok {
		dialOpts = append(dialOpts, extra...)
	}

	conn, err := grpc.NewClient(cfg.Target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create service:%s target:%s gRPC client err:%w", cfg.Service, cfg.Target, err)
	}

	return &Service{
		cfg:  cfg,
		conn: conn,
	}, nil
}

// Invoke 调用该服务下的方法。
// method 可以是方法名，如 SayHello），也可以是完整 gRPC 路径，如 "/Hello.Greeter/SayHello"
func (s *Service) Invoke(ctx context.Context, method string, req, resp proto.Message, opts ...grpc.CallOption) error {
	fullMethod := method
	if !strings.HasPrefix(method, "/") {
		fullMethod = fmt.Sprintf("/%s/%s", s.FullServiceName(), method)
	}

	// 合并metadata
	ctx = mergeMetadata(ctx, s.cfg.Metadata)
	return s.conn.Invoke(ctx, fullMethod, req, resp, opts...)
}

// Conn 返回该服务底层的 gRPC 连接。
func (s *Service) Conn() *grpc.ClientConn {
	return s.conn
}

// Config 返回该服务的配置。
func (s *Service) Config() ServiceConfig {
	if s.cfg == nil {
		return ServiceConfig{}
	}

	return *s.cfg
}

// Close 关闭该服务的连接。
func (s *Service) Close() error {
	if s.conn == nil {
		return nil
	}

	return s.conn.Close()
}

// Target 返回下游目标地址。
func (s *Service) Target() string {
	if s.cfg == nil {
		return ""
	}

	return s.cfg.Target
}

// Name 返回服务逻辑名。
func (s *Service) Name() string {
	if s.cfg == nil {
		return ""
	}

	return s.cfg.Name
}

// FullServiceName 返回service_name
func (s *Service) FullServiceName() string {
	return s.cfg.fullServiceName()
}
