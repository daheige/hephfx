package micro

import (
	"google.golang.org/grpc"

	"github.com/daheige/hephfx/micro/gclient"
)

// RegisterGRPCConnections gRPC client connections manage
// Deprecated: Note that this function is deprecated. Please use gclient.RegisterGRPCConnections instead.
func RegisterGRPCConnections(key string, conn *grpc.ClientConn) {
	gclient.RegisterGRPCConnections(key, conn)
}

// CloseGRPCConnections When the program exits, after the gRPC client call is completed,
// it is generally necessary to close it in the main method of the main.go file.
// It needs to be called use defer gclient.CloseGRPCConnections() method.
// Deprecated: Note that this function is deprecated. Please use gclient.CloseGRPCConnections instead.
func CloseGRPCConnections() {
	gclient.CloseGRPCConnections()
}
