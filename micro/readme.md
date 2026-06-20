# micro

`micro` 是 hephfx 框架中负责 gRPC 服务封装与 HTTP Gateway 代理的核心模块。它基于 `google.golang.org/grpc` 与 `grpc-gateway/v2` 构建，提供统一的微服务启动、优雅停机、拦截器扩展、监控埋点等能力，帮助开发者以较少的配置快速搭建生产级 gRPC/REST 服务。

## 目录

- [核心特性](#核心特性)
- [架构设计](#架构设计)
  - [运行模式](#运行模式)
- [快速开始](#快速开始)
  - [仅启动 gRPC 服务](#仅启动-grpc-服务)
  - [gRPC 与 HTTP Gateway 共享端口](#grpc-与-http-gateway-共享端口)
  - [gRPC 与 HTTP Gateway 独立端口](#grpc-与-http-gateway-独立端口)
- [Option 选项说明](#option-选项说明)
- [核心模块说明](#核心模块说明)
  - [Service](#service)
  - [gRPC 拦截器与中间件](#grpc-拦截器与中间件)
  - [HTTP Gateway 与路由](#http-gateway-与路由)
  - [连接管理](#连接管理)
  - [上下文与元数据工具](#上下文与元数据工具)
- [许可证](#许可证)

## 核心特性

- **统一的服务启动入口**：通过 `micro.NewService` 创建 `*Service`，一行代码即可启动 gRPC 服务。
- **多模式启动**：支持仅启动 gRPC、gRPC + HTTP Gateway 独立端口、gRPC 与 HTTP Gateway 共享端口三种模式。
- **HTTP Gateway 代理**：集成 `grpc-gateway/v2`，可将 RESTful 请求反向代理到 gRPC 服务，支持自定义路由、错误处理与 Metadata 注入。
- **丰富的拦截器生态**：
  - 内置 panic 恢复（`recovery`）拦截器；
  - 内置请求访问日志（`requestInterceptor`）拦截器，自动生成 `x-request-id` 并记录耗时；
  - 内置请求校验（`validator`）拦截器，配合 `validator_gen` 插件可自动生成校验逻辑；
  - 内置 Prometheus 监控拦截器，自动注册 `ServerMetrics`。
- **拦截器可扩展**：支持自定义 `Unary` / `Stream` 拦截器，也支持通过 `grpc.ServerOption` 注入更多原生选项。
- **优雅停机**：监听 `SIGINT/SIGTERM/SIGHUP/SIGQUIT` 等信号，gRPC 与 HTTP 服务均支持优雅关闭，并支持自定义停机函数。
- **可观测性对接**：与 `logger`、`monitor` 模块无缝配合，可快速接入结构化日志、Prometheus 指标与 pprof。
- **连接统一管理**：提供 `RegisterGRPCConnections` / `CloseGRPCConnections`，方便统一关闭 gRPC 客户端连接。

## 架构设计

```text
┌─────────────────────────────────────────────────────────────┐
│                         micro 模块层                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │   Service    │  │  Interceptor │  │  HTTP Gateway    │  │
│  │  gRPC Server │  │ Recovery/Log │  │ grpc-gateway/v2  │  │
│  │  GracefulStop│  │ Validator/   │  │ Mux / Routes /   │  │
│  │  ShutdownHook│  │ Prometheus   │  │ ErrorHandler     │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
   │  纯 gRPC    │     │ gRPC + HTTP │     │  共享端口   │
   │   Server    │     │  独立端口   │     │   (h2c)     │
   └─────────────┘     └─────────────┘     └─────────────┘
```

### 运行模式

`micro.NewService` 支持三种运行模式：

1. **仅启动 gRPC 服务**：不开启 HTTP Gateway，只监听 gRPC 端口。
2. **gRPC 与 HTTP Gateway 独立端口**：gRPC 与 HTTP Gateway 分别监听不同端口，便于独立部署或网络隔离。
3. **gRPC 与 HTTP Gateway 共享端口**：基于 `h2c` 在同一端口上同时提供 gRPC 与 HTTP/REST 服务，减少端口暴露，简化部署。

## 快速开始

### 仅启动 gRPC 服务

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/daheige/hephfx/example/pb"
    "github.com/daheige/hephfx/micro"
)

type GreeterServer struct {
    pb.UnimplementedGreeterServer
}

func (s *GreeterServer) SayHello(ctx context.Context, req *pb.HelloReq) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: fmt.Sprintf("hello,%s", req.Name)}, nil
}

func main() {
    s := micro.NewService(
        "0.0.0.0:50051",
        micro.WithLogger(micro.LoggerFunc(log.Printf)),
        micro.WithShutdownTimeout(5*time.Second),
        micro.WithEnablePrometheus(),        // Prometheus 拦截器
        micro.WithEnableRequestValidator(),  // 请求校验拦截器
        micro.WithShutdownFunc(func() {
            time.Sleep(3 * time.Second)
            log.Println("grpc server shutdown")
        }),
    )

    pb.RegisterGreeterServer(s.GRPCServer, &GreeterServer{})

    if err := s.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### gRPC 与 HTTP Gateway 共享端口

```go
s := micro.NewService(
    "0.0.0.0:50051",
    micro.WithEnableGRPCShareAddress(),
    micro.WithHandlerFromEndpoints(pb.RegisterGreeterHandlerFromEndpoint),
    micro.WithLogger(micro.LoggerFunc(log.Printf)),
    micro.WithShutdownTimeout(5*time.Second),
    micro.WithEnablePrometheus(),
    micro.WithEnableRequestValidator(),
)

pb.RegisterGreeterServer(s.GRPCServer, &GreeterServer{})

if err := s.Run(); err != nil {
    log.Fatal(err)
}
```

HTTP 请求示例：

```shell
curl 'http://localhost:50051/v1/say/daheige'
```

返回：

```json
{"message":"hello,daheige"}
```

### gRPC 与 HTTP Gateway 独立端口

```go
s := micro.NewService(
    "0.0.0.0:50051",
    micro.WithGRPCHTTPAddress("0.0.0.0:8080"),
    micro.WithHandlerFromEndpoints(pb.RegisterGreeterHandlerFromEndpoint),
    micro.WithLogger(micro.LoggerFunc(log.Printf)),
    micro.WithShutdownTimeout(5*time.Second),
    micro.WithEnablePrometheus(),
    micro.WithEnableRequestValidator(),
)

pb.RegisterGreeterServer(s.GRPCServer, &GreeterServer{})

if err := s.Run(); err != nil {
    log.Fatal(err)
}
```

## Option 选项说明

| Option | 说明 |
| --- | --- |
| `WithLogger(logger Logger)` | 设置日志输出器，默认不输出日志。 |
| `WithRecovery(f func())` | 自定义 goroutine recover 处理函数。 |
| `WithShutdownFunc(f func())` | 注册服务优雅停机后的回调函数。 |
| `WithShutdownTimeout(timeout time.Duration)` | 设置停机超时时间，默认 `5s`。 |
| `WithInterruptSignals(signal ...os.Signal)` | 追加需要监听的退出信号。 |
| `WithGRPCServerOption(serverOption ...grpc.ServerOption)` | 追加原生 gRPC `ServerOption`。 |
| `WithUnaryInterceptor(interceptor ...grpc.UnaryServerInterceptor)` | 追加自定义 Unary 拦截器。 |
| `WithStreamInterceptor(interceptor ...grpc.StreamServerInterceptor)` | 追加自定义 Stream 拦截器。 |
| `WithEnableRequestAccess()` | 开启请求访问日志拦截器，自动记录请求方法与耗时。 |
| `WithEnablePrometheus()` | 开启 Prometheus 监控拦截器并自动注册 `ServerMetrics`。 |
| `WithServerMetricsOptions(opts ...gPrometheus.ServerMetricsOption)` | 自定义 Prometheus `ServerMetrics` 选项。 |
| `WithEnableRequestValidator()` | 开启请求校验拦截器，需配合 `validator_gen` 插件使用。 |
| `WithGRPCNetwork(network string)` | 设置 gRPC 监听网络类型，如 `tcp`/`tcp4`/`tcp6`，默认 `tcp`。 |
| `WithEnableHTTPGateway()` | 显式开启 HTTP Gateway。 |
| `WithGRPCHTTPAddress(addr string)` | 设置 HTTP Gateway 监听地址，如 `0.0.0.0:8080`。 |
| `WithEnableGRPCShareAddress()` | gRPC 与 HTTP Gateway 共享同一端口。 |
| `WithHandlerFromEndpoints(h ...HandlerFromEndpoint)` | 注册 `grpc-gateway` 生成的 Handler。 |
| `WithMuxOption(muxOption ...gRuntime.ServeMuxOption)` | 追加 `ServeMux` 选项。 |
| `WithRoutes(routes ...Route)` | 添加 HTTP Gateway 自定义路由。 |
| `WithGRPCEndpointDialOptions(dialOption ...grpc.DialOption)` | 设置 Gateway 反向代理到 gRPC 时的 Dial 选项。 |
| `WithGRPCHTTPServer(server *http.Server)` | 自定义 HTTP Server 实例。 |
| `WithGRPCHTTPHandler(h HTTPHandlerFunc)` | 自定义 HTTP Handler，可集成 Gin/chi/gorilla/mux 等路由。 |
| `WithGRPCHTTPErrorHandler(errorHandler gRuntime.ErrorHandlerFunc)` | 自定义 HTTP Gateway 错误处理函数。 |
| `WithEnableDefaultProtoJSON(b bool)` | 是否启用默认的 protojson `ServeMuxOption`，默认开启。 |

## 核心模块说明

### Service

`Service` 是 `micro` 模块的核心结构，内部封装了：

- `GRPCServer`：原生 `*grpc.Server`，业务代码通过 `pb.RegisterXxxServer(s.GRPCServer, ...)` 注册服务实现。
- gRPC 监听地址、网络类型、信号监听、停机超时等运行参数。
- HTTP Gateway 相关的 `ServeMux`、路由、错误处理器、HTTP Server 等。

`NewService(address string, opts ...Option)` 负责初始化默认配置、安装拦截器、创建 gRPC Server、注册反射服务并构建 Gateway 相关组件。`Run()` 根据配置选择对应的启动模式。

### gRPC 拦截器与中间件

`micro` 默认已安装 `go-grpc-middleware/v2` 的 `recovery` 拦截器，可将 panic 转换为 gRPC 错误。同时支持通过 Option 启用以下能力：

- **请求访问日志**：`WithEnableRequestAccess()` 会注入 `requestInterceptor`，自动注入/读取 `x-request-id`、记录客户端 IP、方法名与耗时。
- **请求校验**：`WithEnableRequestValidator()` 会注入 `validator` 拦截器，业务接口需实现 `Validate()` 方法。
- **Prometheus**：`WithEnablePrometheus()` 会注入 `ServerMetrics` 拦截器并注册到默认 Prometheus Registry。
- **自定义拦截器**：通过 `WithUnaryInterceptor` 与 `WithStreamInterceptor` 可追加任意原生拦截器。

### HTTP Gateway 与路由

`micro` 基于 `grpc-gateway/v2` 提供 HTTP 代理能力：

- 通过 `WithHandlerFromEndpoints` 注册由 `protoc-gen-grpc-gateway` 生成的 Handler。
- 通过 `WithRoutes` 添加纯 HTTP 路由，例如健康检查：

```go
micro.WithRoutes(micro.Route{
    Method: "GET",
    Path:   "/healthz",
    Handler: func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"code":0,"message":"Ok"}`))
    },
})
```

- 通过 `WithGRPCHTTPHandler` 可用 Gin/chi/gorilla/mux 等框架接管 Gateway 请求，例如：

```go
micro.WithGRPCHTTPHandler(func(mux *runtime.ServeMux) http.Handler {
    router := gin.Default()
    router.NoRoute(func(c *gin.Context) {
        mux.ServeHTTP(c.Writer, c.Request)
    })
    return router
})
```

### 连接管理

`conn.go` 提供全局 gRPC 客户端连接管理：

```go
// 注册连接，key 不允许重复
micro.RegisterGRPCConnections("user-service", conn)

// main 函数退出前统一关闭
defer micro.CloseGRPCConnections()
```

### 上下文与元数据工具

`gutils.go` 提供与 gRPC 上下文、Metadata 相关的辅助函数：

| 函数 | 说明 |
| --- | --- |
| `GetGRPCClientIP(ctx)` | 从 gRPC 上下文中获取客户端 IP。 |
| `IncomingMD(ctx)` | 获取 incoming Metadata。 |
| `OutgoingMD(ctx)` | 获取 outgoing Metadata。 |
| `GetStringFromMD(md, key)` | 从 Metadata 中获取字符串值。 |
| `SetCtxValue(ctx, key, val)` | 向 context 中写入键值。 |
| `NewContext(ctx)` | 创建带 `x-request-id` 的新上下文。 |
| `NewTimeoutContext(ctx, timeout)` | 创建带超时的上下文。 |
| `NewRPCContext(ctx, m ...)` | 创建用于下游 gRPC 调用的 outgoing 上下文，自动传递 `x-request-id`。 |
| `GetRPCRequestID(ctx)` | 获取/生成请求 ID。 |
| `SetRPCRequestID(ctx)` | 将请求 ID 写入 incoming Metadata。 |

## 许可证

本项目采用 [MIT License](../LICENSE) 开源协议。

Copyright (c) 2025 heige
