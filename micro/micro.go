package micro

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"time"

	gPrometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	gRecovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	gValidator "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"
	gRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/daheige/hephfx/ctxkeys"
	"github.com/daheige/hephfx/gutils"
)

// HandlerFromEndpoint is the callback that the caller should implement
// to steps to reverse-proxy the HTTP/1 requests to gRPC
// handlerFromEndpoint http gw endPoint
// automatically dials to "endpoint" and closes the connection when "ctx" gets done.
type HandlerFromEndpoint func(ctx context.Context, mux *gRuntime.ServeMux,
	endpoint string, opts []grpc.DialOption) error

// HTTPHandlerFunc is the http middleware handler function.
type HTTPHandlerFunc func(*gRuntime.ServeMux) http.Handler

// AnnotatorFunc is the annotator function is for injecting metadata from http request into gRPC context
type AnnotatorFunc func(context.Context, *http.Request) metadata.MD

// Service gRPC microservice struct
type Service struct {
	// gRPC service settings
	GRPCServer          *grpc.Server                   // gRPC server
	gRPCAddress         string                         // grpc service address,eg:ip:port
	gRPCNetwork         string                         // the gRPC network must be "tcp", "tcp4", "tcp6"
	recovery            func()                         // goroutine exec recover catch stack
	shutdownFunc        func()                         // exec shutdown func after service exit
	shutdownTimeout     time.Duration                  // shutdown wait time,default:5s
	interruptSignals    []os.Signal                    // interrupt signal
	streamInterceptors  []grpc.StreamServerInterceptor // gRPC steam interceptor
	unaryInterceptors   []grpc.UnaryServerInterceptor  // gRPC server interceptor
	serverOptions       []grpc.ServerOption            // gRPC server options
	enableRequestAccess bool                           // gRPC request log config

	// gRPC request validator
	// note: it needs to be used with the validator-gen plugin,
	// for specific usage, refer to the example.
	enableRequestValidator bool

	enablePrometheus     bool // gRPC prometheus monitor
	serverMetricsOptions []gPrometheus.ServerMetricsOption

	logger Logger // logger interface entry

	// gRPC HTTP gateway settings
	enableHTTPGateway bool // enable http gateway,default:false
	// note:If you need to start the HTTP gateway service, you must set this parameter
	handlerFromEndpoints []HandlerFromEndpoint // http gw endpoint

	mux                     *gRuntime.ServeMux        // gRPC http gateway runtime serverMux
	muxOptions              []gRuntime.ServeMuxOption // gRPC HTTP server mux options
	gRPCEndpointDialOptions []grpc.DialOption         // gRPC http gateway dail options
	routes                  []Route                   // gRPC http custom router rules
	gRPCHTTPServer          *http.Server              // gRPC http server
	gRPCHTTPAddress         string                    // gRPC http gateway address,eg:0.0.0.0:8080
	gRPCHTTPHandler         func(*gRuntime.ServeMux) http.Handler
	gRPCHTTPErrorHandler    gRuntime.ErrorHandlerFunc // gRPC http gateway error handler
	enableGRPCShareAddress  bool                      // gRPC server and gRPC http gateway start on one port
	annotators              []AnnotatorFunc           // for injecting metadata from http request into gRPC context
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
		// NewServerMetrics returns a new ServerMetrics object that has server interceptor methods.
		// NOTE: Remember to register ServerMetrics object by using prometheus registry
		// e.g. prometheus.MustRegister(myServerMetrics).
		serverMetrics := gPrometheus.NewServerMetrics(s.serverMetricsOptions...)
		prometheus.MustRegister(serverMetrics)

		s.streamInterceptors = append(s.streamInterceptors, serverMetrics.StreamServerInterceptor())
		s.unaryInterceptors = append(s.unaryInterceptors, serverMetrics.UnaryServerInterceptor())
	}

	// gRPC server options
	s.serverOptions = append(s.serverOptions,
		grpc.ChainStreamInterceptor(s.streamInterceptors...),
		grpc.ChainUnaryInterceptor(s.unaryInterceptors...),
	)

	s.GRPCServer = grpc.NewServer(
		s.serverOptions...,
	)

	// register reflection service on gRPC server.
	reflection.Register(s.GRPCServer)

	// check if the service is started at the same address
	if s.gRPCHTTPAddress != "" {
		if s.gRPCAddress == s.gRPCHTTPAddress {
			s.enableGRPCShareAddress = true
		}

		// the starting address is different
		s.enableGRPCShareAddress = false
		s.enableHTTPGateway = true
	}

	// log.Println("grpc address", s.gRPCAddress, "http gateway address", s.gRPCHTTPAddress)
	if s.enableGRPCShareAddress {
		s.gRPCHTTPAddress = s.gRPCAddress
		s.enableHTTPGateway = true
		// default http server config
		if s.gRPCHTTPServer == nil {
			s.gRPCHTTPServer = &http.Server{
				ReadHeaderTimeout: 5 * time.Second,  // read header timeout
				ReadTimeout:       5 * time.Second,  // read request timeout
				WriteTimeout:      10 * time.Second, // write timeout
				IdleTimeout:       20 * time.Second, // tcp idle time
			}
		}
	}

	// grpc http gateway default config
	if s.enableHTTPGateway {
		// default dial option is using insecure connection
		if len(s.gRPCEndpointDialOptions) == 0 {
			// Deprecated: use WithTransportCredentials and insecure.NewCredentials()
			// instead. Will be supported throughout 1.x.
			// s.gRPCEndpointDialOptions = append(s.gRPCEndpointDialOptions, grpc.WithInsecure())
			// so use grpc.WithTransportCredentials(insecure.NewCredentials()) as default grpc.DialOption
			s.gRPCEndpointDialOptions = append(
				s.gRPCEndpointDialOptions,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
		}

		// default grpc http gateway handler error
		if s.gRPCHTTPErrorHandler == nil {
			s.gRPCHTTPErrorHandler = gRuntime.DefaultHTTPErrorHandler
		}

		// init gateway mux
		// apply default marshal option and error handler for mux options
		s.muxOptions = append(s.muxOptions, defaultMuxOption, gRuntime.WithErrorHandler(s.gRPCHTTPErrorHandler))

		// init annotators
		for _, annotator := range s.annotators {
			s.muxOptions = append(s.muxOptions, gRuntime.WithMetadata(annotator))
		}

		// create http gateway server mux
		s.mux = gRuntime.NewServeMux(s.muxOptions...)

		if s.gRPCHTTPHandler == nil {
			// default grpc http server handler
			s.gRPCHTTPHandler = defaultGRPCHTTPHandler
		}
	}

	return s
}

// Run start service
func (s *Service) Run() error {
	// start gRPC server and gRPC http gateway server on one port
	if s.enableGRPCShareAddress {
		return s.startUseOneAddress()
	}

	// only start gRPC server
	if !s.enableHTTPGateway {
		return s.startGRPCService()
	}

	// start grpc and http gateway server on different port
	return s.startTwoServices()
}

// Run start gRPC service
func (s *Service) startGRPCService() error {
	// intercept interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, s.interruptSignals...)
	// channels to receive error
	errChan := make(chan error, 1)

	// only start gRPC server
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
		s.stopGRPCServer()
		return nil
	}
}

// startTwoServices starts the microservice with listening on the ports
// start grpc gateway and http server on different port
func (s *Service) startTwoServices() error {
	if s.gRPCHTTPAddress == s.gRPCAddress {
		return errors.New("gRPC server and gRPC http gateway address are the same")
	}

	// default http server config
	if s.gRPCHTTPServer == nil {
		s.gRPCHTTPServer = &http.Server{
			ReadHeaderTimeout: 5 * time.Second,  // read header timeout
			ReadTimeout:       5 * time.Second,  // read request timeout
			WriteTimeout:      10 * time.Second, // write timeout
			IdleTimeout:       20 * time.Second, // tcp idle time
		}
	}

	// intercept interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, s.interruptSignals...)

	// channels to receive error
	errChan1 := make(chan error, 1)
	errChan2 := make(chan error, 1)

	// start gRPC server
	go func() {
		defer s.recovery()

		s.logger.Printf("Starting gPRC server listening on %s\n", s.gRPCAddress)
		errChan1 <- s.startGRPCServer()
	}()

	// start gRPC HTTP gateway server
	go func() {
		defer s.recovery()

		s.logger.Printf("Starting gRPC http gateway server listening on %s\n", s.gRPCHTTPAddress)
		errChan2 <- s.startGRPCGateway()
	}()

	// wait for context cancellation or shutdown signal
	select {
	// if gRPC server fail to start
	case err := <-errChan1:
		return err
	// if http server fail to start
	case err := <-errChan2:
		return err
	// if we received an interrupt signal
	case sig := <-sigChan:
		s.logger.Printf("Interrupt signal received: %v\n", sig)
		s.stopTwoServices()
		return nil
	}
}

// stops the microservice gracefully.
func (s *Service) stopTwoServices() {
	// disable keep-alive on existing connections
	s.gRPCHTTPServer.SetKeepAlivesEnabled(false)

	// gracefully stop gRPC server first
	s.GRPCServer.GracefulStop()

	// gracefully stop http server
	s.httpServerShutdown()
	s.shutdownFunc() // exec shutdown function
}

// start gRPC service and gRPC http gateway on one address
func (s *Service) startUseOneAddress() error {
	// intercept interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, s.interruptSignals...)
	// channels to receive error
	errChan := make(chan error, 1)

	// grpc and grpc http gateway start on one port
	go func() {
		defer s.recovery()

		s.logger.Printf("Starting gRPC http gateway and gRPC server listening on %s\n", s.gRPCHTTPAddress)
		errChan <- s.startGRPCAndHTTPServer()
	}()

	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan: // Block until we receive our signal.
		s.logger.Printf("interrupt signal received: %v\n", sig)
		s.stopGRPCAndHTTPServer()
		return nil
	}
}

func (s *Service) startGRPCAndHTTPServer() error {
	if s.gRPCAddress == "" {
		return fmt.Errorf("gRPC address is required")
	}

	s.gRPCHTTPAddress = s.gRPCAddress
	ctx := context.Background()
	var err error
	for _, h := range s.handlerFromEndpoints {
		err = h(ctx, s.mux, s.gRPCAddress, s.gRPCEndpointDialOptions)
		if err != nil {
			s.logger.Printf("register handler from endPoint error: %s\n", err.Error())
			return err
		}
	}

	// apply routes
	err = s.appRoutes()
	if err != nil {
		return err
	}

	// http server and h2c handler
	// create an http mux
	httpMux := http.NewServeMux()
	httpMux.Handle("/", s.mux)

	// gRPC HTTP server address
	s.gRPCHTTPServer.Addr = s.gRPCHTTPAddress

	// convert HTTP requests to http2
	s.gRPCHTTPServer.Handler = GRPCHandlerFunc(s.GRPCServer, httpMux)
	return s.gRPCHTTPServer.ListenAndServe()
}

func (s *Service) stopGRPCAndHTTPServer() {
	// disable keep-alive on existing connections
	s.gRPCHTTPServer.SetKeepAlivesEnabled(false)

	// gracefully stop http server
	s.httpServerShutdown()

	// exec shutdown function
	s.shutdownFunc()
}

// httpServerShutdown http gateway server graceful shutdown.
func (s *Service) httpServerShutdown() {
	done := make(chan struct{}, 1)
	ctx, cancel := context.WithTimeout(
		context.Background(),
		s.shutdownTimeout,
	)

	defer cancel()

	// gracefully stop http server
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// if your application should wait for other services
	// to finalize based on context cancellation.
	// gracefully stop http server
	go func() {
		defer s.recovery()
		defer close(done)

		if err := s.gRPCHTTPServer.Shutdown(ctx); err != nil {
			s.logger.Printf("Http server shutdown error: %v", err.Error())
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Printf("Server shutdown ctx cancel error: %v", ctx.Err())
	case <-done:
		s.logger.Printf("Server shutdown success")
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
func (s *Service) stopGRPCServer() {
	done := make(chan struct{}, 1)
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
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

	s.logger.Printf("gRPC server shutdown success")
}

// startGRPCServer start grpc server.
func (s *Service) startGRPCServer() error {
	if s.gRPCNetwork == "" {
		s.gRPCNetwork = "tcp"
	}

	lis, err := net.Listen(s.gRPCNetwork, s.gRPCAddress)
	if err != nil {
		return err
	}

	return s.GRPCServer.Serve(lis)
}

func (s *Service) startGRPCGateway() error {
	// Register http gw handlerFromEndpoint
	ctx := context.Background()
	var err error
	for _, h := range s.handlerFromEndpoints {
		err = h(ctx, s.mux, s.gRPCAddress, s.gRPCEndpointDialOptions)
		if err != nil {
			s.logger.Printf("register handler from endPoint error: %s\n", err.Error())
			return err
		}
	}

	// apply routes
	err = s.appRoutes()
	if err != nil {
		return err
	}

	// http server
	s.gRPCHTTPServer.Addr = s.gRPCHTTPAddress
	s.gRPCHTTPServer.Handler = s.gRPCHTTPHandler(s.mux)
	return s.gRPCHTTPServer.ListenAndServe()
}

func (s *Service) appRoutes() error {
	for _, route := range s.routes {
		if !strings.HasPrefix(route.Path, "/") {
			route.Path = "/" + route.Path
		}

		err := s.mux.HandlePath(route.Method, route.Path, func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
			route.Handler(w, r)
		})

		if err != nil {
			s.logger.Printf("add router error:%s,current method:%s path:%s invalid", err.Error(),
				route.Method, route.Path)
			return err
		}
	}

	return nil
}

// refer: https://github.com/golang/protobuf/blob/v1.4.3/jsonpb/encode.go#L30
var defaultMuxOption = gRuntime.WithMarshalerOption(gRuntime.MIMEWildcard, &gRuntime.JSONPb{})

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

// defaultGRPCHTTPHandler is the default gRPC http handler which does nothing.
func defaultGRPCHTTPHandler(mux *gRuntime.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	})
}

// GRPCHandlerFunc uses the standard library h2c to convert http requests to http2
// In this way, you can co-exist with go grpc and http services, and use one port
// to provide both grpc services and http services.
// In June 2018, the golang.org/x/net/http2/h2c standard library representing the "h2c"
// logo was officially merged in, and since then we can use the official standard library (h2c)
// This standard library implements the unencrypted mode of HTTP/2,
// so we can use the standard library to provide both HTTP/1.1 and HTTP/2 functions on the same port
// The h2c.NewHandler method has been specially processed, and h2c.NewHandler will return an http.handler
// The main internal logic is to intercept all h2c traffic, then hijack and redirect it
// to the corresponding handler according to different request traffic types to process.
// If a request is a h2c connection, it's hijacked and redirected to
// s.ServeConn. Otherwise the returned Handler just forwards requests to http.
func GRPCHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor >= 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}
