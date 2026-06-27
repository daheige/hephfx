package bridge

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/status"
)

// ErrServiceNotFound 服务未找到错误。
var ErrServiceNotFound = errors.New("service not found")

// IsNotFound 判断错误是否为服务未找到。
func IsNotFound(err error) bool {
	return errors.Is(err, ErrServiceNotFound)
}

// GRPCError 从 gRPC 错误中提取状态信息。
func GRPCError(err error) (*status.Status, bool) {
	if err == nil {
		return nil, false
	}

	st, ok := status.FromError(err)
	return st, ok
}

func serviceNotFound(name string) error {
	return fmt.Errorf("%s:%w", name, ErrServiceNotFound)
}
