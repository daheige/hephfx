package consul

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/daheige/hephfx/gutils"
	"github.com/daheige/hephfx/hestia"
)

var _ hestia.Registry = (*consulRegistry)(nil)

type consulRegistry struct {
	client          *consulapi.Client
	ttl             string
	deregisterAfter string
	keepaliveCtx    context.Context
	keepaliveCancel context.CancelFunc
	validateAddress bool
	prefix          string
}

// NewRegistry create a consul Registry instance
func NewRegistry(endpoints []string, opts ...Option) (hestia.Registry, error) {
	opt := &Options{
		endpoints:                      endpoints,
		dialTimeout:                    5 * time.Second,
		ttl:                            "10s",
		deregisterCriticalServiceAfter: "1m",
		prefix:                         "/hestia/registry-consul",
	}

	for _, o := range opts {
		o(opt)
	}

	client, err := newConsulClient(opt)
	if err != nil {
		return nil, err
	}

	r := &consulRegistry{
		client:          client,
		ttl:             opt.ttl,
		deregisterAfter: opt.deregisterCriticalServiceAfter,
		validateAddress: opt.validateAddress,
		prefix:          opt.prefix,
	}

	return r, nil
}

// Register service instance register
func (r *consulRegistry) Register(ctx context.Context, s *hestia.Service) error {
	if s.InstanceID == "" {
		s.InstanceID = gutils.Uuid()
	}

	if r.validateAddress {
		address, err := hestia.Resolve(s.Address)
		if err != nil {
			return fmt.Errorf("failed to resolve address:%v error:%v", s.Address, err)
		}
		s.Address = address
	}

	if s.Weight == 0 {
		s.Weight = 100
	}
	s.Healthy = true

	host, port, err := splitHostPort(s.Address)
	if err != nil {
		return fmt.Errorf("failed to split address %s: %v", s.Address, err)
	}

	tags := buildTags(s, r.prefix)
	meta := buildMeta(s)
	checkID := buildCheckID(s.InstanceID)

	// Single call with embedded check — matches Rust convention
	serviceReg := &consulapi.AgentServiceRegistration{
		ID:      s.InstanceID,
		Name:    s.Name,
		Address: host,
		Port:    port,
		Tags:    tags,
		Meta:    meta,
		Check: &consulapi.AgentServiceCheck{
			CheckID:                        checkID,
			Name:                           fmt.Sprintf("%s TTL check", s.Name),
			TTL:                            r.ttl,
			DeregisterCriticalServiceAfter: r.deregisterAfter,
		},
	}

	if err := r.client.Agent().ServiceRegister(serviceReg); err != nil {
		return fmt.Errorf("consul register service %s error: %v", s.Name, err)
	}

	// Start keepalive goroutine
	r.stopKeepalive()
	r.keepaliveCtx, r.keepaliveCancel = context.WithCancel(context.Background())
	go r.keepalive(r.keepaliveCtx, checkID)

	log.Printf("consul register service:%s version:%s instanceID:%s host:%s port:%d checkID:%s success\n",
		s.Name, s.Version, s.InstanceID, host, port, checkID)
	return nil
}

// Deregister the service goes offline when the application exit
func (r *consulRegistry) Deregister(ctx context.Context, s *hestia.Service) error {
	if s.Name == "" {
		return fmt.Errorf("missing service name in Deregister")
	}

	r.stopKeepalive()

	checkID := buildCheckID(s.InstanceID)
	// deregister the check first
	if err := r.client.Agent().CheckDeregister(checkID); err != nil {
		log.Printf("consul deregister check %s warning: %v", checkID, err)
	}

	// then deregister the service
	err := r.client.Agent().ServiceDeregister(s.InstanceID)
	if err != nil {
		return fmt.Errorf("consul deregister service %s error: %v", s.Name, err)
	}

	s.Healthy = false
	log.Printf("consul deregister service:%s version:%s instanceID:%s success\n",
		s.Name, s.Version, s.InstanceID)
	return nil
}

// String returns the name of the registry
func (r *consulRegistry) String() string {
	return "consul"
}

func (r *consulRegistry) stopKeepalive() {
	if r.keepaliveCancel != nil {
		r.keepaliveCancel()
	}
}

func (r *consulRegistry) keepalive(ctx context.Context, checkID string) {
	dur, err := time.ParseDuration(r.ttl)
	if err != nil {
		log.Printf("consul keepalive parse ttl %s error: %v", r.ttl, err)
		return
	}
	// TTL heartbeat interval: TTL/2 to match Rust convention
	interval := dur / 2
	if interval < time.Second {
		interval = time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("consul keepalive started checkID:%s ttl:%s interval:%v", checkID, r.ttl, interval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("consul keepalive stopped checkID:%s: %v", checkID, ctx.Err())
			// best-effort: mark check as failing on shutdown
			_ = r.client.Agent().UpdateTTL(checkID, "service shutting down", "failing")
			return
		case <-ticker.C:
			if err := r.client.Agent().UpdateTTL(checkID, "healthy", "passing"); err != nil {
				log.Printf("consul keepalive update TTL checkID:%s error: %v", checkID, err)
			}
		}
	}
}

func buildMeta(s *hestia.Service) map[string]string {
	meta := make(map[string]string, len(s.Metadata))
	for k, v := range s.Metadata {
		meta[k] = fmt.Sprintf("%v", v)
	}
	return meta
}

func buildTags(s *hestia.Service, prefix string) []string {
	var tags []string
	if prefix != "" {
		tags = append(tags, "prefix:"+normalizePrefix(prefix))
	}
	if s.Version != "" {
		tags = append(tags, "version:"+s.Version)
	}
	tags = append(tags, "protocol:"+string(s.Protocol))
	tags = append(tags, "instance_id:"+s.InstanceID)
	return tags
}

func buildCheckID(instanceID string) string {
	return "service:" + instanceID
}

func splitHostPort(address string) (host string, port int, err error) {
	h, pStr, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}
	p, err := strconv.Atoi(pStr)
	if err != nil {
		return "", 0, err
	}
	return h, p, nil
}

func newConsulClient(opt *Options) (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()
	if len(opt.endpoints) > 0 {
		addr := opt.endpoints[0]
		addr = strings.TrimPrefix(addr, "http://")
		addr = strings.TrimPrefix(addr, "https://")
		config.Address = addr
	}
	if opt.token != "" {
		config.Token = opt.token
	}

	return consulapi.NewClient(config)
}
