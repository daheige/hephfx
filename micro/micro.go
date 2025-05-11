package micro

import (
	"context"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"time"

	gRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	gValidator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	gPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/daheige/hephfx/ctxkeys"
	"github.com/daheige/hephfx/gutils"
)

// Service gRPC microservice struct
type Service struct {
	GRPCServer         *grpc.Server                   // gRPC server
	gRPCAddress        string                         // grpc service address,eg:ip:port
	gRPCNetwork        string                         // the gRPC network must be "tcp", "tcp4", "tcp6"
	recovery           func()                         // goroutine exec recover catch stack
	shutdownFunc       func()                         // exec shutdown func after service exit
	shutdownTimeout    time.Duration                  // shutdown wait time
	interruptSignals   []os.Signal                    // interrupt signal
	streamInterceptors []grpc.StreamServerInterceptor // gRPC steam interceptor
	unaryInterceptors  []grpc.UnaryServerInterceptor  // gRPC server interceptor
	serverOptions      []grpc.ServerOption            // gRPC server options

	enableRequestAccess bool // gRPC request log config

	// gRPC request validator
	// note: it needs to be used with the validator-gen plugin,
	// for specific usage, refer to the example.
	enableRequestValidator bool

	enablePrometheus bool   // gRPC prometheus monitor
	logger           Logger // logger interface entry
}

// NewService create a grpc service instance
func NewService(address string, opts ...Option) *Service {
	s := defaultService()
	s.gRPCAddress = address

	// app option functions
	for _, o := range opts {
		o(s)
	}

	// install validator interceptor.
	if s.enableRequestValidator {
		s.streamInterceptors = append(s.streamInterceptors, gValidator.StreamServerInterceptor())
		s.unaryInterceptors = append(s.unaryInterceptors, gValidator.UnaryServerInterceptor())
	}

	// install request interceptor
	if s.enableRequestAccess {
		s.unaryInterceptors = append(s.unaryInterceptors, s.requestInterceptor)
	}

	// install prometheus interceptor
	if s.enablePrometheus {
		s.streamInterceptors = append(s.streamInterceptors, gPrometheus.StreamServerInterceptor)
		s.unaryInterceptors = append(s.unaryInterceptors, gPrometheus.UnaryServerInterceptor)
	}

	// gRPC server options
	s.serverOptions = append(s.serverOptions,
		grpc.ChainStreamInterceptor(s.streamInterceptors...),
		grpc.ChainUnaryInterceptor(s.unaryInterceptors...),
	)

	s.GRPCServer = grpc.NewServer(
		s.serverOptions...,
	)

	return s
}

// Run start gRPC service
func (s *Service) Run() error {
	// intercept interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, s.interruptSignals...)
	// channels to receive error
	errChan := make(chan error, 1)

	// start gRPC server
	go func() {
		defer s.recovery()

		s.logger.Printf("start gPRC server listening on %s\n", s.gRPCAddress)
		errChan <- s.startGRPCServer()
	}()

	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan: // Block until we receive our signal.
		s.logger.Printf("interrupt signal received: %v\n", sig)
		s.stopServer()
		return nil
	}
}

// requestInterceptor request interceptor to record basic information of the request
func (s *Service) requestInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (reply interface{}, err error) {
	t := time.Now()
	md := IncomingMD(ctx) // get request metadata
	requestID := GetStringFromMD(md, ctxkeys.XRequestID)
	if requestID == "" {
		requestID = gutils.Uuid()
		md.Set(ctxkeys.XRequestID.String(), requestID)
	}

	defer func() {
		if r := recover(); r != nil {
			// the error format defined by grpc must be used here to return code, desc
			err = status.Errorf(codes.Internal, "%s", "server inner error")
			s.logger.Printf("x-request-id:%s exec panic:%v req:%v reply:%v\n", requestID, r, req, reply)
			s.logger.Printf("x-request-id:%s full stack:%s\n", requestID, string(debug.Stack()))
		}
	}()

	// request ip
	clientIP, _ := GetGRPCClientIP(ctx)

	// exec begin
	s.logger.Printf("exec begin,method:%s x-request-id:%s client-ip:%s\n", info.FullMethod, requestID, clientIP)

	// set request ctx key
	md.Set(ctxkeys.ClientIP.String(), clientIP)
	md.Set(ctxkeys.RequestMethod.String(), info.FullMethod)

	ctx = metadata.NewIncomingContext(ctx, md)

	reply, err = handler(ctx, req)

	// exec end
	ttd := time.Since(t).Milliseconds()
	if err != nil {
		s.logger.Printf("x-request-id:%s trace_error:%s reply:%v exec_time:%v\n", requestID, err.Error(), reply, ttd)
		return nil, err
	}

	s.logger.Printf("exec end,method:%s x-request-id:%s cost time:%vms\n", info.FullMethod, requestID, ttd)

	return reply, err
}

// GetPid gets the process id of server
func (s *Service) GetPid() int {
	return os.Getpid()
}

// stopServer stop the gRPC server gracefully
func (s *Service) stopServer() {
	done := make(chan struct{}, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// graceful exit current service
	go func() {
		defer func() {
			s.recovery()
			close(done)
		}()

		// stoop grpc server
		s.GRPCServer.GracefulStop()

		// exec shutdown function
		s.shutdownFunc()
	}()

	select {
	case <-done:
		s.logger.Printf("stop server done\n")
	case <-ctx.Done():
		s.logger.Printf("stop server context timeout\n")
	}

	s.logger.Printf("grpc server shutdown success")
}

// startGRPCServer start grpc server.
func (s *Service) startGRPCServer() error {
	// register reflection service on gRPC server.
	reflection.Register(s.GRPCServer)

	if s.gRPCNetwork == "" {
		s.gRPCNetwork = "tcp"
	}

	lis, err := net.Listen(s.gRPCNetwork, s.gRPCAddress)
	if err != nil {
		return err
	}

	return s.GRPCServer.Serve(lis)
}

func defaultService() *Service {
	s := &Service{
		gRPCNetwork:        "tcp",
		shutdownTimeout:    5 * time.Second,
		interruptSignals:   interruptSignals,
		streamInterceptors: make([]grpc.StreamServerInterceptor, 0, 20),
		unaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0, 20),
		serverOptions:      make([]grpc.ServerOption, 0, 8),
		logger:             dummyLogger,
	}

	// default shutdown function
	s.shutdownFunc = func() {}

	// goroutine recover catch stack
	s.recovery = func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Printf("exec recover: %v\n", r)
				s.logger.Printf("full stack: %s\n", string(debug.Stack()))
			}
		}()
	}

	// install panic handler which will turn panics into gRPC errors.
	s.streamInterceptors = append(s.streamInterceptors, gRecovery.StreamServerInterceptor())
	s.unaryInterceptors = append(s.unaryInterceptors, gRecovery.UnaryServerInterceptor())

	return s
}
