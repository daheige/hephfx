package consul

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/daheige/hephfx/hestia"
)

var _ hestia.Discovery = (*consulDiscovery)(nil)

type consulDiscovery struct {
	client       *consulapi.Client
	serviceList  map[string][]*hestia.Service
	disableWatch bool
	mu           sync.RWMutex
}

// NewDiscovery create a consul Discovery instance
func NewDiscovery(endpoints []string, opts ...Option) (hestia.Discovery, error) {
	opt := &Options{
		endpoints:    endpoints,
		dialTimeout:  5 * time.Second,
		ttl:          "30s",
		disableWatch: true,
	}

	for _, o := range opts {
		o(opt)
	}

	client, err := newConsulClient(opt)
	if err != nil {
		return nil, err
	}

	d := &consulDiscovery{
		client:       client,
		serviceList:  make(map[string][]*hestia.Service, 20),
		disableWatch: opt.disableWatch,
	}

	return d, nil
}

// GetServices returns a list of instances
func (d *consulDiscovery) GetServices(ctx context.Context, name string, version string) ([]*hestia.Service, error) {
	var (
		services []*hestia.Service
		exist    bool
	)

	if !d.disableWatch {
		d.mu.RLock()
		services, exist = d.serviceList[name]
		d.mu.RUnlock()
	}

	if !exist {
		var err error
		services, err = d.getServicesByName(ctx, name, version)
		if err != nil {
			return nil, err
		}
		if len(services) == 0 {
			return nil, hestia.ErrServicesNotFound
		}

		d.mu.Lock()
		d.serviceList[name] = services
		d.mu.Unlock()

		if !d.disableWatch {
			go d.watch(context.WithoutCancel(ctx), name, version)
		}
	}

	return services, nil
}

// Get returns an available service instance based on the specified service selection strategy.
func (d *consulDiscovery) Get(ctx context.Context, name string, version string,
	strategyHandler ...hestia.StrategyHandler) (*hestia.Service, error) {
	services, err := d.GetServices(ctx, name, version)
	if err != nil {
		return nil, err
	}

	handler := hestia.RoundRobinHandler
	if len(strategyHandler) > 0 && strategyHandler[0] != nil {
		handler = strategyHandler[0]
	}

	return handler(services), nil
}

// String returns discovery name
func (d *consulDiscovery) String() string {
	return "consul"
}

func (d *consulDiscovery) watch(ctx context.Context, name string, version string) {
	d.watchWithCallback(ctx, name, version, func(services []*hestia.Service, err error) {
		if err != nil {
			log.Printf("consul watch services error: %v", err)
			return
		}

		d.mu.Lock()
		d.serviceList[name] = services
		d.mu.Unlock()
	})
}

// watchWithCallback watches the service and invokes callback on every change
// using Consul blocking queries.
func (d *consulDiscovery) watchWithCallback(ctx context.Context, name string, version string,
	callback func([]*hestia.Service, error)) {
	tag := buildVersionFilter(version)
	q := &consulapi.QueryOptions{
		WaitTime: 5 * time.Minute,
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		entries, meta, err := d.client.Health().Service(name, tag, true, q)
		if err != nil {
			callback(nil, fmt.Errorf("consul health service %s error: %v", name, err))
			// On error, wait before retry to avoid hammering the server
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			q.WaitIndex = 0
			continue
		}

		q.WaitIndex = meta.LastIndex
		services := mapToServices(entries)
		callback(services, nil)
	}
}

func (d *consulDiscovery) getServicesByName(ctx context.Context,
	name string, version string) ([]*hestia.Service, error) {
	tag := buildVersionFilter(version)
	q := &consulapi.QueryOptions{}
	entries, _, err := d.client.Health().Service(name, tag, true, q.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("consul health service %s error: %v", name, err)
	}

	return mapToServices(entries), nil
}

// buildVersionFilter returns the version tag used for Consul health filtering.
// Returns empty string if version is empty (no version filter).
func buildVersionFilter(version string) string {
	if version == "" {
		return ""
	}
	return versionTagPrefix + version
}

// mapToServices maps Consul ServiceEntry list to hestia.Service list
func mapToServices(entries []*consulapi.ServiceEntry) []*hestia.Service {
	services := make([]*hestia.Service, 0, len(entries))
	for _, entry := range entries {
		s := mapToService(entry)
		if s != nil {
			services = append(services, s)
		}
	}
	return services
}

// mapToService maps a single Consul ServiceEntry to hestia.Service
func mapToService(entry *consulapi.ServiceEntry) *hestia.Service {
	svc := entry.Service
	meta := svc.Meta

	weight, _ := strconv.ParseUint(meta["weight"], 10, 32)
	if weight == 0 {
		weight = 100
	}

	// Extract version from meta or from tags
	version := meta["version"]

	tags := make(map[string]string)
	metadata := make(map[string]interface{})
	for k, v := range meta {
		if strings.HasPrefix(k, "tag_") {
			tagKey := strings.TrimPrefix(k, "tag_")
			tags[tagKey] = v
		} else if strings.HasPrefix(k, "meta_") {
			metaKey := strings.TrimPrefix(k, "meta_")
			metadata[metaKey] = v
		}
	}

	s := &hestia.Service{
		Network:       getMeta(meta, "network", "tcp"),
		Name:          svc.Service,
		Address:       fmt.Sprintf("%s:%d", svc.Address, svc.Port),
		NamingAddress: meta["naming_address"],
		InstanceID:    svc.ID,
		Version:       version,
		Weight:        uint32(weight),
		Protocol:      hestia.ProtocolType(meta["protocol"]),
		Healthy:       true,
		Created:       meta["created"],
		Metadata:      metadata,
		Tags:          tags,
	}

	return s
}

func getMeta(meta map[string]string, key, defaultVal string) string {
	if v, ok := meta[key]; ok {
		return v
	}
	return defaultVal
}
