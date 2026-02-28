package micro

import (
	"net/http"
	"os"
	"time"

	gPrometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	gRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// Option for grpc service option
type Option func(s *Service)

// WithRecovery service recover func
func WithRecovery(f func()) Option {
	return func(s *Service) {
		s.recovery = f
	}
}

// WithUnaryInterceptor returns an Option to append some unaryInterceptor
func WithUnaryInterceptor(unaryInterceptor ...grpc.UnaryServerInterceptor) Option {
	return func(s *Service) {
		s.unaryInterceptors = append(s.unaryInterceptors, unaryInterceptor...)
	}
}

// WithStreamInterceptor returns an Option to append some streamInterceptor
func WithStreamInterceptor(streamInterceptor ...grpc.StreamServerInterceptor) Option {
	return func(s *Service) {
		s.streamInterceptors = append(s.streamInterceptors, streamInterceptor...)
	}
}

// WithShutdownFunc returns an Option to register a function which will be called when server shutdown
func WithShutdownFunc(f func()) Option {
	return func(s *Service) {
		s.shutdownFunc = f
	}
}

// WithShutdownTimeout returns an Option to set the timeout before the server shutdown abruptly
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(s *Service) {
		s.shutdownTimeout = timeout
	}
}

// WithInterruptSignals returns an Option to append a interrupt signal
func WithInterruptSignals(signal ...os.Signal) Option {
	return func(s *Service) {
		s.interruptSignals = append(s.interruptSignals, signal...)
	}
}

// WithGRPCServerOption returns an Option to append a gRPC server option
func WithGRPCServerOption(serverOption ...grpc.ServerOption) Option {
	return func(s *Service) {
		s.serverOptions = append(s.serverOptions, serverOption...)
	}
}

// WithLogger uses the provided logger
func WithLogger(logger Logger) Option {
	return func(s *Service) {
		s.logger = logger
	}
}

// WithEnableRequestAccess request access log config
func WithEnableRequestAccess() Option {
	return func(s *Service) {
		s.enableRequestAccess = true
	}
}

// WithEnablePrometheus enable prometheus
func WithEnablePrometheus() Option {
	return func(s *Service) {
		s.enablePrometheus = true
	}
}

// WithServerMetricsOptions set prometheus server metrics options
func WithServerMetricsOptions(opts ...gPrometheus.ServerMetricsOption) Option {
	return func(s *Service) {
		s.serverMetricsOptions = append(s.serverMetricsOptions, opts...)
	}
}

// WithEnableRequestValidator set request validator interceptor
func WithEnableRequestValidator() Option {
	return func(s *Service) {
		s.enableRequestValidator = true
	}
}

// WithGRPCNetwork set gRPC start network type
func WithGRPCNetwork(network string) Option {
	return func(s *Service) {
		s.gRPCNetwork = network
	}
}

// WithHandlerFromEndpoints add HandlerFromEndpoint
func WithHandlerFromEndpoints(h ...HandlerFromEndpoint) Option {
	return func(s *Service) {
		s.handlerFromEndpoints = append(s.handlerFromEndpoints, h...)
	}
}

// WithEnableHTTPGateway enable http gateway
func WithEnableHTTPGateway() Option {
	return func(s *Service) {
		s.enableHTTPGateway = true
	}
}

// WithMuxOption returns an Option to append a mux option
func WithMuxOption(muxOption ...gRuntime.ServeMuxOption) Option {
	return func(s *Service) {
		s.muxOptions = append(s.muxOptions, muxOption...)
	}
}

// WithRoutes adds additional routes
func WithRoutes(routes ...Route) Option {
	return func(s *Service) {
		s.routes = append(s.routes, routes...)
	}
}

// WithGRPCEndpointDialOptions returns an Option to append a gRPC dial option
func WithGRPCEndpointDialOptions(dialOption ...grpc.DialOption) Option {
	return func(s *Service) {
		s.gRPCEndpointDialOptions = append(s.gRPCEndpointDialOptions, dialOption...)
	}
}

// WithGRPCHTTPServer returns an Option to set the http server
func WithGRPCHTTPServer(server *http.Server) Option {
	return func(s *Service) {
		s.gRPCHTTPServer = server
	}
}

// WithGRPCHTTPAddress set gRPC HTTP Address,eg: 0.0.0.0:8080
func WithGRPCHTTPAddress(addr string) Option {
	return func(s *Service) {
		s.gRPCHTTPAddress = addr
	}
}

// WithGRPCHTTPHandler returns an Option to set gRPC HTTP Server Handler
func WithGRPCHTTPHandler(h HTTPHandlerFunc) Option {
	return func(s *Service) {
		s.gRPCHTTPHandler = h
	}
}

// WithGRPCHTTPErrorHandler returns an Option to set the errorHandler
func WithGRPCHTTPErrorHandler(errorHandler gRuntime.ErrorHandlerFunc) Option {
	return func(s *Service) {
		s.gRPCHTTPErrorHandler = errorHandler
	}
}

// WithEnableGRPCShareAddress returns an Option to set the enableGRPCShareAddress
func WithEnableGRPCShareAddress() Option {
	return func(s *Service) {
		s.enableGRPCShareAddress = true
	}
}
