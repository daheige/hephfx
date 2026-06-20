package hestia

import (
	"errors"
	"math/rand"
	"sync/atomic"
	"time"
)

// ErrServicesNotFound 服务列表为空
var ErrServicesNotFound = errors.New("services not found")

// Service 服务基本信息
type Service struct {
	// network name of the network (for example, "tcp", "udp")
	Network string `json:"network"`

	// 服务名字
	Name string `json:"name"`

	// 服务地址，一般来说由host:port组成
	Address string `json:"address"`

	// 命名服务的地址，例如：k8s的user.local.svc
	NamingAddress string `json:"naming_address"`

	// 服务的唯一标识，例如uuid字符串
	InstanceID string `json:"instance_id"`

	// 当前版本
	Version string `json:"version"`

	// 创建时间
	Created string `json:"created"`

	// 服务的其他元信息
	// Metadata map[string]string{} `json:"metadata"`
	Metadata map[string]interface{} `json:"metadata"`

	// 其他标签信息
	Tags map[string]string `json:"tags"`
}

// StrategyHandler service selection strategy
type StrategyHandler func(services []*Service) *Service

// roundRobinCounter is used by RoundRobinHandler to select the next instance.
var roundRobinCounter uint64

// RoundRobinHandler returns the next service instance in round-robin order.
func RoundRobinHandler(services []*Service) *Service {
	if len(services) == 0 {
		return nil
	}

	idx := atomic.AddUint64(&roundRobinCounter, 1) - 1
	return services[idx%uint64(len(services))]
}

// RandomHandler returns a random service instance.
func RandomHandler(services []*Service) *Service {
	if len(services) == 0 {
		return nil
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return services[r.Intn(len(services))]
}
