// Package bridge 提供基于 bridge_services 配置的下游 gRPC 服务客户端 SDK。
//
// 它读取 app.yaml 中的 bridge_services 列表，为每个服务创建独立的 gRPC 连接，
// 并允许按逻辑服务名直接调用下游方法，无需业务方手动管理连接。
//
// 配置文件示例：
//
//	bridge_services:
//	  - name: uc-svc
//	    target: uc.cluster.local:8080
//	    service: "uc.UserService"
//	  - name: rbac-svc
//	    target: rbac.cluster.local:8080
//	    service: "rbac.RBAC"
//	  - name: greeter-svc
//	    target: greeter.cluster.local:8080
//	    service: "Hello.Greeter"
//	    metadata:
//	      x-service: "greeter"
//
// 基本用法：
//
//	cfg, err := bridge.LoadConfig("./app.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	client, err := bridge.NewClient(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
//	defer cancel()
//
//	req := &pb.HelloReq{Name: "daheige"}
//	var resp pb.HelloReply
//	if err := client.Invoke(ctx, "greeter-svc", "SayHello", req, &resp); err != nil {
//	    log.Fatal(err)
//	}
//
// 如需使用完整 gRPC 路径调用：
//
//	svc, err := client.Service("greeter-svc")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if err := svc.Invoke(ctx, "/Hello.Greeter/SayHello", req, &resp); err != nil {
//	    log.Fatal(err)
//	}
package bridge
