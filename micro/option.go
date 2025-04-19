package micro

import (
	"os"
	"time"

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

// WithEnableRequestValidator set request validator
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
