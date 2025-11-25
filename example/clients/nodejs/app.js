let services = require('./pb/hello_grpc_pb.js');
let messages = require('./pb/hello_pb.js');
let grpc = require('@grpc/grpc-js');

// 创建请求对象
let request = new messages.HelloReq();
request.setName("daheige");

// 创建grpc client
let client = new services.GreeterClient(
    'localhost:50051',
    grpc.credentials.createInsecure()
);

// 调用grpc微服务的方法
client.sayHello(request, function(err, data) {
    if (err) {
        console.error("user login error: ",err);
        return;
    }

    console.log("response data:",data);
    console.log("message: ",data.getMessage());
});
