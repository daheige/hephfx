package micro

import (
	"log"

	"google.golang.org/grpc"
)

// the grpc client makes a connection call to the connection management.
var gRPCConnections = map[string]*grpc.ClientConn{}

// RegisterGRPCConnections register grpc client connection
// If there are multiple grpc client calls in a project, it is recommended to manage them uniformly.
// Before the main function of the program exits, all of them can be closed by
// calling the defer micro.CloseGRPCConnections() method,
// thereby releasing the grpc connections.
func RegisterGRPCConnections(key string, conn *grpc.ClientConn) {
	_, ok := gRPCConnections[key]
	if ok {
		log.Fatalf("grpc connection key:%s already exists", key)
	}

	gRPCConnections[key] = conn
}

// CloseGRPCConnections When the program exits, after the grpc client call is completed,
// it is generally necessary to close it in the main method of the main.go file.
// It needs to be called use defer micro.CloseGRPCConnections() method.
func CloseGRPCConnections() {
	for k := range gRPCConnections {
		err := gRPCConnections[k].Close()
		if err != nil {
			log.Printf("grpc connection key:%s close error:%v\n", k, err)
		}
	}
}
