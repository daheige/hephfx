# Hephfx

A microservice framework for gRPC.

- 名字源自希腊神话中的 Hephaestus（赫菲斯托斯）‌，象征创造力、工艺与自动化‌，作为火神与锻造之神，其形象常与工具制造、工业效率相关联‌。
- 因此，采用这个名字的前面4个字母和fx组合后的名字`hephfx`代表特效`effects`或功能扩展`function extension`，表示工具的实用性与模块化能力‌。
- 该框架旨在帮助开发人员快速学习和上手gRPC微服务，降低微服务接入和学习成本，让开发人员能够更好地聚焦业务逻辑开发而设计。

# 目录

- [核心特性](#核心特性)
- [架构设计](#架构设计)
  - [运行模式](#运行模式)
- [主要目录](#主要目录)
- [环境准备](#环境准备)
- [快速开始](#快速开始)
  - [启动 gRPC 服务](#启动-grpc-服务)
  - [启动 gRPC HTTP Gateway 代理](#启动-grpc-http-gateway-代理)
- [组件库说明和使用](#组件库说明和使用)
  - [micro](#micro)
  - [hestia](#hestia)
  - [logger](#logger)
  - [monitor](#monitor)
  - [settings](#settings)
  - [ctxkeys / gutils](#ctxkeys--gutils)
- [许可证](#许可证)

## 核心特性

- **gRPC 服务封装**：基于 `google.golang.org/grpc` 封装服务端启动流程，支持纯 gRPC、gRPC + HTTP Gateway 双端口、以及单端口共存三种启动模式。
- **HTTP Gateway 代理**：集成 `grpc-gateway/v2`，支持通过同一端口或独立端口暴露 RESTful API，并提供自定义路由、错误处理、Metadata 注入等扩展能力。
- **中间件生态**：内置 panic 恢复、请求日志、请求校验、Prometheus 监控等拦截器，同时支持自定义 unary/stream 拦截器。
- **服务注册与发现**：`hestia` 模块提供 `Registry` / `Discovery` 抽象接口，内置基于 etcd 的实现，支持 lease 心跳保活、版本隔离、watch 监听以及 gRPC resolver。
- **可观测性**：`monitor` 模块集成 Prometheus 指标与 Go pprof，提供 HTTP 接口 `/metrics` 与 `/debug/pprof`。
- **日志能力**：`logger` 模块基于 zap 封装，支持 JSON/文本输出、日志切割、文件/终端同时输出以及 Sentry 错误上报。
- **配置读取**：`settings` 模块基于 viper 抽象统一的配置读取接口，支持文件热加载。
- **工具方法**：`gutils` 提供 UUID、MD5、随机数、panic 堆栈捕获等常用工具函数；`ctxkeys` 提供上下文键名常量。

## 架构设计

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                              hephfx 框架层                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌──────────────────┐  │
│  │   micro    │  │   logger   │  │   monitor  │  │ settings/ctxkeys │  │
│  │ gRPC 服务  │  │  日志输出  │  │ metrics/   │  │  配置与上下文    │  │
│  │ + Gateway  │  │  sentry    │  │ pprof      │  │  辅助            │  │
│  └────────────┘  └────────────┘  └────────────┘  └──────────────────┘  │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                         hestia 服务治理层                        │   │
│  │  Registry 接口 ──► etcdRegistry（lease 保活、注销）              │   │
│  │  Discovery 接口 ──► etcdDiscovery（Get/GetServices/watch）       │   │
│  │  gRPC Resolver ──► etcd:///service_name/version                  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │   etcd / 扩展注册中心 │
                         └─────────────────────┘
```

### 运行模式

`micro.NewService` 支持三种运行模式：

1. **纯 gRPC 服务**：仅监听 gRPC 端口。
2. **gRPC + HTTP Gateway 独立端口**：gRPC 与 HTTP Gateway 分别监听不同端口。
3. **gRPC 与 HTTP Gateway 共享端口**：基于 `h2c` 协议在同一端口上同时提供 gRPC 与 HTTP/REST 服务。

## 主要目录

```ini
./
├── LICENSE                       # MIT 开源协议
├── README.md / readme.md         # 项目说明
├── go.mod / go.sum               # Go 模块依赖
├── grpc-server.png               # gRPC 服务端运行截图
├── grpc-http-proxy.png           # HTTP Gateway 运行截图
├── example                       # gRPC 实战 demo
│   ├── bin                       # 编译产物目录
│   ├── clients                   # 多语言客户端示例
│   │   ├── go                    # Go 客户端示例
│   │   └── nodejs                # Node.js 客户端示例
│   ├── cmd                       # 可执行程序入口
│   │   ├── gateway               # HTTP Gateway 代理示例
│   │   └── rpc                   # gRPC 服务端示例
│   ├── internal                  # 业务内部实现
│   │   └── interfaces            # 接口层实现
│   ├── pb                        # protobuf 生成的 Go 代码
│   ├── protos                    # .proto 源文件
│   │   └── google                # google api proto 依赖
│   └── tools                     # 工具脚本
│       └── validator_gen         # 校验码生成工具
├── ctxkeys                       # 上下文键名常量（request_id、client_ip 等）
├── gutils                        # 通用工具函数（UUID、MD5、随机数、堆栈捕获等）
├── hestia                        # 服务注册与发现，基于 etcd 实现
│   ├── etcd                      # etcd 注册中心、发现、resolver 实现
│   ├── discovery.go              # Discovery 接口
│   ├── registry.go               # Registry 接口
│   ├── service_entity.go         # Service 实体定义
│   └── netaddr.go                # 本机地址解析
├── logger                        # 基于 zap 的日志组件
│   └── example                   # logger 使用示例
│       └── sentry                # Sentry 上报示例
├── micro                         # gRPC 微服务封装
│   ├── micro.go                  # Service 核心实现
│   ├── option.go                 # 启动配置选项
│   ├── router.go                 # HTTP Gateway 自定义路由
│   ├── conn.go                   # gRPC 连接相关
│   ├── logger.go                 # 日志接口适配
│   ├── signals.go                # 信号处理
│   └── md_test.go / gutils.go    # 辅助与测试
├── monitor                       # 服务监控指标与 Go pprof
│   ├── gpprof                    # pprof HTTP 服务封装
│   ├── monitor.go                # HTTP 监控中间件
│   ├── prometheus.go             # Prometheus 初始化
│   └── monitor_test.go           # 测试
└── settings                      # 配置文件读取（基于 viper）
    ├── config.go                 # Config 接口
    ├── viper_config.go           # viper 实现
    ├── option.go                 # 配置选项
    └── settings_test.go          # 测试
```

## 环境准备

1. 进入 https://go.dev/dl/ 官方网站，根据系统安装不同的go版本，这里推荐在linux或mac系统上面安装go。
2. 设置 GOPROXY

```shell
go env -w GOPROXY=https://goproxy.cn,direct
```

3. 安装 protoc 工具

- mac系统安装方式如下：

```shell
brew install protobuf
```

- linux系统安装方式如下：

```shell
# Reference: https://grpc.io/docs/protoc-installation/
PB_REL="https://github.com/protocolbuffers/protobuf/releases"
curl -LO $PB_REL/download/v3.15.8/protoc-3.15.8-linux-x86_64.zip
unzip -o protoc-3.15.8-linux-x86_64.zip -d $HOME/.local
export PATH=~/.local/bin:$PATH # Add this to your `~/.bashrc`.
protoc --version
libprotoc 3.15.8
```

4. 安装 gRPC 相关的 Go 工具链

```shell
# go gRPC tools
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/daheige/validator_gen@latest

#This will place four binaries in your $GOBIN;
#    protoc-gen-grpc-gateway
#    protoc-gen-openapiv2
#    protoc-gen-go
#    protoc-gen-go-grpc

# google api link:https://github.com/googleapis/googleapis

# protoc inject tag
go install github.com/favadi/protoc-go-inject-tag@latest
```

## 快速开始

参考：[example](example) 或者 https://github.com/daheige/hephfx-micro-svc

### 启动 gRPC 服务

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/daheige/hephfx/example/pb"
    "github.com/daheige/hephfx/logger"
    "github.com/daheige/hephfx/micro"
    "github.com/daheige/hephfx/monitor"
)

func main() {
    logger.Default(
        logger.WithStdout(true),
        logger.WithJsonFormat(true),
    )

    s := micro.NewService(
        "0.0.0.0:50051",
        micro.WithEnableGRPCShareAddress(),
        micro.WithHandlerFromEndpoints(pb.RegisterGreeterHandlerFromEndpoint),
        micro.WithLogger(micro.LoggerFunc(log.Printf)),
        micro.WithShutdownTimeout(5*time.Second),
        micro.WithEnablePrometheus(),
        micro.WithEnableRequestValidator(),
        micro.WithRoutes(micro.Route{
            Method: "GET",
            Path:   "/healthz",
            Handler: func(w http.ResponseWriter, r *http.Request) {
                w.Write([]byte(`{"code":0,"message":"Ok"}`))
            },
        }),
    )

    // metrics: http://localhost:8090/metrics
    // pprof:  http://localhost:8090/debug/pprof
    monitor.InitMonitor(8090)

    pb.RegisterGreeterServer(s.GRPCServer, &GreeterServer{})
    if err := s.Run(); err != nil {
        log.Fatal(err)
    }
}

type GreeterServer struct {
    pb.UnimplementedGreeterServer
}

func (s *GreeterServer) SayHello(ctx context.Context, req *pb.HelloReq) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: fmt.Sprintf("hello,%s", req.Name)}, nil
}
```

服务端运行效果如下：

![example](grpc-server.png)

metrics and pprof

- metrics访问地址：http://localhost:8090/metrics
- pprof访问地址：http://localhost:8090/debug/pprof

### 启动 gRPC HTTP Gateway 代理

参考：[example/cmd/gateway](example/cmd/gateway)

运行效果如下：

![grpc-http-proxy](grpc-http-proxy.png)

metrics and pprof

- metrics访问地址：http://localhost:9091/metrics
- pprof访问地址：http://localhost:9091/debug/pprof

## 组件库说明和使用

### micro

`micro` 是框架的核心模块，负责 gRPC 服务的创建、启动、优雅停机以及 HTTP Gateway 的注册。

```go
s := micro.NewService(
    "0.0.0.0:50051",                                    // gRPC 监听地址
    micro.WithGRPCHTTPAddress("0.0.0.0:8080"),          // HTTP Gateway 独立端口
    // micro.WithEnableGRPCShareAddress(),                // 与 gRPC 共享端口
    micro.WithHandlerFromEndpoints(pb.RegisterGreeterHandlerFromEndpoint),
    micro.WithEnableRequestAccess(),                      // 开启请求日志
    micro.WithEnablePrometheus(),                         // 开启 Prometheus 拦截器
    micro.WithEnableRequestValidator(),                   // 开启请求校验
    micro.WithShutdownTimeout(5*time.Second),
)
```
micro实战参考：
- https://github.com/daheige/hephfx-micro-svc
- https://github.com/daheige/hephfx/tree/main/example

常用配置项见 [micro/option.go](micro/option.go)。micro更多用法查看：https://github.com/daheige/hephfx/tree/main/micro

### hestia

`hestia` 提供服务注册与发现能力，内置 etcd 实现。

- 服务端注册

```go
registry, _ := etcd.NewRegistry([]string{"http://127.0.0.1:12379"})
registry.Register(ctx, &hestia.Service{
    Network: "tcp",
    Name:    "my-service",
    Address: ":8080",
    Version: "v1",
})
```

- 客户端发现

```go
discovery, _ := etcd.NewDiscovery([]string{"http://127.0.0.1:12379"})
svc, _ := discovery.Get(ctx, "my-service", "v1")
```

- gRPC resolver

```go
etcd.RegisterEtcdResolver(discovery)
conn, _ := grpc.NewClient(
    "etcd:///my-service/v1",
    grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
```

更多用法参考 [hestia/readme.md](hestia/readme.md)。

### logger

基于 zap 的日志组件，支持 JSON 格式化、文件输出、日志切割和 Sentry 错误上报。

```go
logger.Default(
    logger.WithJsonFormat(true),
    logger.WithLogLevel(zap.InfoLevel),
    logger.WithStdout(true),
    logger.WithEnableSentry(true),
    logger.WithSentryLevel(zap.ErrorLevel),
)

logger.Info(ctx, "hello", "key", "value")
logger.Error(ctx, "something wrong", "err", err)
```

更多用法参考 [logger/readme.md](logger/readme.md)。

### monitor

集成 Prometheus 指标与 Go pprof，内置 HTTP 请求总量与耗时统计。

```go
// 端口 8090：http://localhost:8090/metrics
// 端口 8090：http://localhost:8090/debug/pprof
monitor.InitMonitor(8090)

// Web 服务可额外注册 web_request_total/web_request_duration_seconds
monitor.InitMonitor(9091, true)
```

### settings

基于 viper 的配置读取模块，支持 YAML 等格式以及文件变更监听。

```go
cfg, err := settings.Load("./app.yaml")
if err != nil {
    log.Fatal(err)
}

cfg.ReadSection("server", &serverConfig)
```

### ctxkeys / gutils

- `ctxkeys`：定义上下文键常量，避免 key 冲突。
- `gutils`：提供 `Uuid()`、`Md5()`、`RandInt64()`、`CatchStack()` 等工具函数。

```go
requestID := ctxkeys.XRequestID.String()
uuid := gutils.Uuid()
stack := gutils.CatchStack()
```

## 许可证

本项目采用 [MIT License](LICENSE) 开源协议。

Copyright (c) 2025 heige
