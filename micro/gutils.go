package micro

import (
	"context"
	"errors"
	"net"
	"strings"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/daheige/hephfx/ctxkeys"
)

var (
	// ErrInvokeGRPCClientIP get grpc request client ip fail.
	ErrInvokeGRPCClientIP = errors.New("invoke from context failed")

	// ErrPeerAddressNil gRPC peer address is nil.
	ErrPeerAddressNil = errors.New("peer address is nil")
)

// GetGRPCClientIP get client ip address from context
func GetGRPCClientIP(ctx context.Context) (string, error) {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return "", ErrInvokeGRPCClientIP
	}

	if pr.Addr == net.Addr(nil) {
		return "", ErrPeerAddressNil
	}

	addSlice := strings.Split(pr.Addr.String(), ":")

	return addSlice[0], nil
}

// IncomingMD returns metadata.MD from incoming ctx
// get request metadata
// this method is mainly used at the server end to get the relevant metadata data
func IncomingMD(ctx context.Context) metadata.MD {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return metadata.MD{}
	}

	return md
}

// OutgoingMD returns metadata.MD from outgoing ctx
// Use this method when you pass ctx to a downstream service
func OutgoingMD(ctx context.Context) metadata.MD {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return metadata.MD{}
	}

	return md
}

// GetSliceFromMD returns []string from md
func GetSliceFromMD(md metadata.MD, key ctxkeys.CtxKey) []string {
	return md.Get(key.String())
}

// GetStringFromMD returns string from md
func GetStringFromMD(md metadata.MD, key ctxkeys.CtxKey) string {
	values := md.Get(key.String())
	if len(values) > 0 {
		return values[0]
	}

	return ""
}

// SetCtxValue returns ctx when you set key/value into ctx
func SetCtxValue(ctx context.Context, key ctxkeys.CtxKey, val interface{}) context.Context {
	return context.WithValue(ctx, key.String(), val)
}

// GetCtxValue returns ctx when you set key/value into ctx
func GetCtxValue(ctx context.Context, key ctxkeys.CtxKey) interface{} {
	return ctx.Value(key)
}
