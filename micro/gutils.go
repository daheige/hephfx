package micro

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/daheige/hephfx/ctxkeys"
	"github.com/daheige/hephfx/gutils"
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
	return context.WithValue(ctx, key, val)
}

// GetCtxValue returns ctx when you set key/value into ctx
func GetCtxValue(ctx context.Context, key ctxkeys.CtxKey) interface{} {
	return ctx.Value(key)
}

// NewContext create a new context from request,eg:http request
func NewContext(ctx context.Context) context.Context {
	requestID, ok := ctx.Value(ctxkeys.XRequestID).(string)
	if requestID == "" || !ok {
		requestID = gutils.Uuid()
	}

	newCtx := context.WithValue(ctx, ctxkeys.XRequestID, requestID)
	return newCtx
}

// NewTimeoutContext create a new context from request,eg:http request
func NewTimeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	newCtx, cancel := context.WithTimeout(NewContext(ctx), timeout)
	return newCtx, cancel
}

// NewRPCContext new a context for grpc client call
func NewRPCContext(ctx context.Context, m ...map[string]string) context.Context {
	requestID, ok := ctx.Value(ctxkeys.XRequestID).(string)
	if requestID == "" || !ok {
		requestID = gutils.Uuid()
	}

	// set request-id into outgoing ctx metadata
	md := OutgoingMD(ctx)
	md.Set(ctxkeys.XRequestID.String(), requestID)
	if len(m) > 0 {
		for _, meta := range m {
			if len(meta) > 0 {
				for k, v := range meta {
					md.Set(k, v)
				}
			}
		}
	}

	newCtx := metadata.NewOutgoingContext(ctx, md)
	return newCtx
}

// GetRPCRequestID returns request-id from grpc request metadata.FromIncomingContext
func GetRPCRequestID(ctx context.Context) string {
	md := IncomingMD(ctx) // get request metadata
	requestID := GetStringFromMD(md, ctxkeys.XRequestID)
	if requestID == "" {
		requestID = gutils.Uuid()
	}

	return requestID
}

// SetRPCRequestID set request-id into incoming metadata
func SetRPCRequestID(ctx context.Context) metadata.MD {
	md := IncomingMD(ctx) // get request metadata
	requestID := GetStringFromMD(md, ctxkeys.XRequestID)
	if requestID == "" {
		requestID = gutils.Uuid()
		md.Set(ctxkeys.XRequestID.String(), requestID)
	}
	
	return md
}
