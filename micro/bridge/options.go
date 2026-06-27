package bridge

import (
	"time"

	"google.golang.org/grpc"
)

// ClientOptions 创建 Client 时的选项。
type ClientOptions struct {
	options         []grpc.DialOption            // 全局 grpc DialOption
	serviceDialOpts map[string][]grpc.DialOption // 单个服务的 grpc DialOption
	idleTimeout     time.Duration                // 空闲时间，默认30m
	maxCallAttempts int                          // 最大重试次数，默认为3
	serviceConfig   string                       // grpc service 负载均衡策略
}

// ClientOption 用于配置 Client。
type ClientOption func(*ClientOptions)

// WithDialOptions 设置全局 gRPC 拨号选项。
func WithDialOptions(opts ...grpc.DialOption) ClientOption {
	return func(o *ClientOptions) {
		o.options = append(o.options, opts...)
	}
}

// WithIdleTimeout 设置 idleTimeout
func WithIdleTimeout(d time.Duration) ClientOption {
	return func(o *ClientOptions) {
		o.idleTimeout = d
	}
}

// WithMaxCallAttempts 设置 maxCallAttempts
func WithMaxCallAttempts(n int) ClientOption {
	return func(o *ClientOptions) {
		o.maxCallAttempts = n
	}
}

// WithServiceConfig 设置 grpc service 负载均衡策略
func WithServiceConfig(cfg string) ClientOption {
	return func(o *ClientOptions) {
		o.serviceConfig = cfg
	}
}

// WithServiceDialOpts 设置单个服务的 grpc DialOption
func WithServiceDialOpts(name string, opts ...grpc.DialOption) ClientOption {
	return func(o *ClientOptions) {
		if o.serviceDialOpts == nil {
			o.serviceDialOpts = make(map[string][]grpc.DialOption)
		}
		if o.serviceDialOpts[name] == nil {
			o.serviceDialOpts[name] = make([]grpc.DialOption, 0, 10) // 预分配
		}

		o.serviceDialOpts[name] = append(o.serviceDialOpts[name], opts...)
	}
}
