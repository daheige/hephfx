# example
    hephfx/micro实战
# gen code
执行如下命令实现go代码生成
```shell
sh bin/go-generate.sh
```

# start running
1. 先运行命令`go run cmd/rpc/main.go`启动服务端。
2. 接着执行`go run clients/go/main.go`运行客户端。

# grpc gateway
1. 需要在proto文件添加如下核心配置
```protobuf
import "google/api/annotations.proto";

// Greeter service 定义开放调用的服务
service Greeter {
    rpc SayHello (HelloReq) returns (HelloReply){
        option (google.api.http) = {
            get: "/v1/say/{id}"
        };
    };

    rpc Info (InfoReq) returns (InfoReply){
        option (google.api.http) = {
            get: "/v1/info/{name}"
        };
    };
}
```
2. 执行`go run cmd/gateway/main.go`即可（启动之前，需要先启动rpc服务端）。

# grpc grpcurl tools
- grpcurl工具主要用于grpcurl请求，可以快速查看grpc proto定义以及调用grpc service定义的方法。
- grpcurl参考地址：https://github.com/fullstorydev/grpcurl

1. 安装grpcurl工具
```shell
brew install grpcurl
```
如果你本地安装了golang，那可以直接运行如下命令，安装grpcurl工具
```shell
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```
2. 验证rs-rpc service启动的效果
```shell
# 50051 是qa-svc grpc微服务的端口
grpcurl -plaintext 127.0.0.1:50051 list
```

执行上面的命令，输出结果如下：
```ini
Hello.Greeter
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
```
3. 查看proto文件定义的所有方法
```shell
grpcurl -plaintext 127.0.0.1:50051 describe Hello.Greeter
```
输出结果如下：
```ini
Hello.Greeter is a service:
service Greeter {
  rpc Info ( .Hello.InfoReq ) returns ( .Hello.InfoReply ) {
    option (.google.api.http) = { get: "/v1/info/{name}" };
  }
  rpc SayHello ( .Hello.HelloReq ) returns ( .Hello.HelloReply ) {
    option (.google.api.http) = { get: "/v1/say/{id}" };
  }
}
```
4. 查看请求参数定义
```shell
grpcurl -plaintext 127.0.0.1:50051 describe Hello.HelloReq
```
输出结果如下：
```ini
Hello.HelloReq is a message:
message HelloReq {
  int64 id = 1;
}
```
5. 请求grpc方法
```shell
grpcurl -d '{"id":1}' -plaintext 127.0.0.1:50051 Hello.Greeter.SayHello
```
返回结果如下：
```json
{
  "name": "heph-fx,id: 1",
  "message": "hello world"
}
```
