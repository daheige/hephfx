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
	client        *consulapi.Client
	serviceList   map[string][]*hestia.Service
	disableWatch  bool
	prefix        string
	watchInterval time.Duration
	mu            sync.RWMutex
}

// NewDiscovery create a consul Discovery instance
func NewDiscovery(endpoints []string, opts ...Option) (hestia.Discovery, error) {
	opt := &Options{
		endpoints:     endpoints,
		dialTimeout:   5 * time.Second,
		ttl:           "10s",
		prefix:        "/hestia/registry-consul",
		disableWatch:  true,
		watchInterval: 30 * time.Second,
	}

	for _, o := range opts {
		o(opt)
	}

	client, err := newConsulClient(opt)
	if err != nil {
		return nil, err
	}

	d := &consulDiscovery{
		client:        client,
		serviceList:   make(map[string][]*hestia.Service, 20),
		disableWatch:  opt.disableWatch,
		prefix:        opt.prefix,
		watchInterval: opt.watchInterval,
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

// watchWithCallback periodically polls the service and invokes callback on every tick.
func (d *consulDiscovery) watchWithCallback(ctx context.Context, name string, version string,
	callback func([]*hestia.Service, error)) {
	ticker := time.NewTicker(d.watchInterval)
	defer ticker.Stop()

	// Fetch once immediately on start
	d.pollAndCallback(ctx, name, version, callback)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.pollAndCallback(ctx, name, version, callback)
		}
	}
}

func (d *consulDiscovery) pollAndCallback(ctx context.Context, name string, version string,
	callback func([]*hestia.Service, error)) {
	services, err := d.getServicesByName(ctx, name, version)
	if err != nil {
		callback(nil, err)
		return
	}
	callback(services, nil)
}

func (d *consulDiscovery) getServicesByName(ctx context.Context,
	name string, version string) ([]*hestia.Service, error) {
	tag := buildVersionFilter(version)
	q := &consulapi.QueryOptions{}
	entries, _, err := d.client.Health().Service(name, tag, true, q.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("consul health service %s error: %v", name, err)
	}

	entries = filterByPrefix(entries, d.prefix)
	return mapToServices(entries), nil
}

// buildVersionFilter returns the version tag used for Consul health filtering.
func buildVersionFilter(version string) string {
	if version == "" {
		return ""
	}
	return "version:" + version
}

// tagValue finds the first tag starting with the given prefix and returns
// the remainder after the prefix. Returns empty string if not found.
func tagValue(tags []string, prefix string) string {
	for _, t := range tags {
		if strings.HasPrefix(t, prefix) {
			return t[len(prefix):]
		}
	}
	return ""
}

// parseWeight returns the weight from the "weight:" tag, default 100.
func parseWeight(tags []string) uint32 {
	s := tagValue(tags, "weight:")
	if s == "" {
		return 100
	}
	w, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 100
	}
	return uint32(w)
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

// filterByPrefix returns only the entries whose service tags contain the
// matching prefix tag. When prefix is empty, all entries are returned.
func filterByPrefix(entries []*consulapi.ServiceEntry, prefix string) []*consulapi.ServiceEntry {
	if prefix == "" {
		return entries
	}

	target := "prefix:" + normalizePrefix(prefix)
	filtered := entries[:0]
	for _, entry := range entries {
		for _, tag := range entry.Service.Tags {
			if tag == target {
				filtered = append(filtered, entry)
				break
			}
		}
	}

	return filtered
}

// mapToService maps a single Consul ServiceEntry to hestia.Service
func mapToService(entry *consulapi.ServiceEntry) *hestia.Service {
	svc := entry.Service

	// Read fields from tags
	prefix := tagValue(svc.Tags, "prefix:")
	version := tagValue(svc.Tags, "version:")
	protocolStr := tagValue(svc.Tags, "protocol:")
	instanceID := tagValue(svc.Tags, "instance_id:")
	if instanceID == "" {
		instanceID = svc.ID
	}
	network := tagValue(svc.Tags, "network:")
	if network == "" {
		network = "tcp"
	}
	weight := parseWeight(svc.Tags)
	created := tagValue(svc.Tags, "created:")
	namingAddress := tagValue(svc.Tags, "naming_address:")

	// Node address fallback when service address is empty
	host := svc.Address
	if host == "" {
		host = entry.Node.Address
	}

	// Build metadata from svc.Meta directly
	metadata := make(map[string]interface{}, len(svc.Meta))
	for k, v := range svc.Meta {
		metadata[k] = v
	}

	// Build tags from Consul service tags
	tags := map[string]string{
		"prefix": prefix,
	}
	if version != "" {
		tags["version"] = version
	}
	if protocolStr != "" {
		tags["protocol"] = protocolStr
	}
	if instanceID != "" {
		tags["instance_id"] = instanceID
	}
	if network != "" {
		tags["network"] = network
	}
	if created != "" {
		tags["created"] = created
	}
	if namingAddress != "" {
		tags["naming_address"] = namingAddress
	}

	return &hestia.Service{
		Network:       network,
		Name:          svc.Service,
		Address:       fmt.Sprintf("%s:%d", host, svc.Port),
		NamingAddress: namingAddress,
		InstanceID:    instanceID,
		Version:       version,
		Weight:        weight,
		Protocol:      hestia.ProtocolType(protocolStr),
		Healthy:       true,
		Created:       created,
		Metadata:      metadata,
		Tags:          tags,
	}
}
