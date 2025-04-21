package hestia

import (
	"errors"
	"math/rand"
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

// RoundRobinHandler returns a random service instance.
func RoundRobinHandler(services []*Service) *Service {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 生成一个随机索引
	randomIndex := r.Intn(len(services))
	// 通过随机索引获取元素
	service := services[randomIndex]
	return service
}
