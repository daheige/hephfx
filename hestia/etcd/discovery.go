package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/daheige/hephfx/hestia"
)

type etcdDiscovery struct {
	client *clientv3.Client
	prefix string
}

// NewDiscovery create a registry interface instance
func NewDiscovery(endpoints []string, opts ...Option) (hestia.Discovery, error) {
	opt := &Options{
		endpoints:   endpoints,
		dialTimeout: 5 * time.Second,
		prefix:      "hestia/registry-etcd",
	}

	for _, o := range opts {
		o(opt)
	}

	client, err := newEtcdClient(opt)
	if err != nil {
		return nil, err
	}

	e := &etcdDiscovery{
		client: client,
		prefix: opt.prefix,
	}

	return e, nil
}

// GetServices returns a list of instances
// After we obtain the service instance, we can get the currently available service instance
// from the service list according to different strategies.
func (e *etcdDiscovery) GetServices(name string) ([]*hestia.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	key := fmt.Sprintf("%s/%s", e.prefix, name)
	// log.Println("key: ", key)
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, hestia.ErrServicesNotFound
	}

	// 获取所有的服务实例列表
	services := make([]*hestia.Service, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		serviceEntry := &hestia.Service{}
		err = json.Unmarshal(kv.Value, serviceEntry)
		if err != nil {
			log.Printf("unmarshal service failed,error:%v", err)
			continue
		}

		services = append(services, serviceEntry)
	}

	return services, nil
}

// Get returns an available service instance based on the specified service selection strategy.
// the selection strategy is RoundRobinHandler
func (e *etcdDiscovery) Get(name string, strategyHandler ...hestia.StrategyHandler) (*hestia.Service, error) {
	services, err := e.GetServices(name)
	if err != nil {
		return nil, err
	}

	var handler = hestia.RoundRobinHandler
	if len(strategyHandler) > 0 && strategyHandler[0] != nil {
		handler = strategyHandler[0]
	}

	service := handler(services)
	return service, nil
}

// String returns discovery name
func (e *etcdDiscovery) String() string {
	return "etcd"
}
