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
