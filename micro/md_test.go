package micro

import (
	"context"
	"log"
	"testing"
)

func TestNewRPCContext(t *testing.T) {
	// mock rpc client call,new a context
	ctx := NewRPCContext(context.Background(), map[string]string{
		"key": "abc",
	}, map[string]string{
		"key2": "hello",
	})

	// print outgoing md
	md := OutgoingMD(ctx)
	log.Println("md:", md)
}
