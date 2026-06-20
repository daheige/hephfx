package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/daheige/hephfx/hestia"
)

var _ hestia.Discovery = (*etcdDiscovery)(nil)

type etcdDiscovery struct {
	client       *clientv3.Client
	prefix       string
	serviceList  map[string][]*hestia.Service
	disableWatch bool // disable watch
	mu           sync.RWMutex
}

// NewDiscovery create a registry interface instance
func NewDiscovery(endpoints []string, opts ...Option) (hestia.Discovery, error) {
	opt := &Options{
		endpoints:    endpoints,
		dialTimeout:  5 * time.Second,
		prefix:       "/hestia/registry-etcd",
		disableWatch: true, // disable watch
	}

	for _, o := range opts {
		o(opt)
	}

	client, err := newEtcdClient(opt)
	if err != nil {
		return nil, err
	}

	e := &etcdDiscovery{
		client:       client,
		prefix:       opt.prefix,
		serviceList:  make(map[string][]*hestia.Service, 20),
		disableWatch: opt.disableWatch,
	}

	e.prefix = strings.TrimPrefix(e.prefix, "/")
	e.prefix = strings.TrimSuffix(e.prefix, "/")
	e.prefix = fmt.Sprintf("/%s", e.prefix) // 格式为/services

	return e, nil
}

// GetServices returns a list of instances
// After we obtain the service instance, we can get the currently available service instance
// from the service list according to different strategies.
func (e *etcdDiscovery) GetServices(ctx context.Context, name string, version string) ([]*hestia.Service, error) {
	var (
		services []*hestia.Service
		exist    bool
	)
	if !e.disableWatch {
		e.mu.RLock()
		services, exist = e.serviceList[name]
		e.mu.RUnlock()
	}

	if !exist {
		var err error
		services, err = e.getServicesByName(ctx, name, version)
		if err != nil {
			return nil, err
		}
		if len(services) == 0 {
			return nil, hestia.ErrServicesNotFound
		}

		e.mu.Lock()
		e.serviceList[name] = services
		e.mu.Unlock()

		if !e.disableWatch {
			go e.watch(context.WithoutCancel(ctx), name, version)
		}
	}

	return services, nil
}

// Get returns an available service instance based on the specified service selection strategy.
// the selection strategy is RoundRobinHandler
func (e *etcdDiscovery) Get(ctx context.Context, name string, version string, strategyHandler ...hestia.StrategyHandler) (*hestia.Service, error) {
	services, err := e.GetServices(ctx, name, version)
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

// listen services change
func (e *etcdDiscovery) watch(ctx context.Context, name string, version string) {
	key := e.discoveryKey(name, version)
	e.watchWithCallback(ctx, name, version, func(services []*hestia.Service, err error) {
		if err != nil {
			log.Printf("reload etcd prefix:%s services error:%v", key, err)
			return
		}

		e.mu.Lock()
		e.serviceList[name] = services
		e.mu.Unlock()
	})
}

// watchWithCallback watches the service prefix and invokes callback on every change.
// It is package-private so the etcd resolver can consume real-time updates without
// exposing watch semantics in the public Discovery API.
func (e *etcdDiscovery) watchWithCallback(ctx context.Context, name string, version string,
	callback func([]*hestia.Service, error)) {
	key := e.discoveryKey(name, version)
	// log.Printf("watch etcd prefix:%v\n", key)
	respChan := e.client.Watch(ctx, key, clientv3.WithPrefix())
	for resp := range respChan {
		for _, event := range resp.Events {
			switch event.Type {
			case clientv3.EventTypePut, clientv3.EventTypeDelete:
				services, err := e.getServicesByName(ctx, name, version)
				callback(services, err)
			}
		}
	}
}

func (e *etcdDiscovery) discoveryKey(name, version string) string {
	var key string
	if version == "" {
		key = fmt.Sprintf("%s/%s/", e.prefix, name)
	} else {
		key = fmt.Sprintf("%s/%s/%s/", e.prefix, name, version)
	}

	return key
}

func (e *etcdDiscovery) getServicesByName(ctx context.Context, name string, version string) ([]*hestia.Service, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	key := e.discoveryKey(name, version)
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
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
		if serviceEntry.Healthy {
			services = append(services, serviceEntry)
		}
	}

	return services, nil
}
