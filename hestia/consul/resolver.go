package consul

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc/resolver"

	"github.com/daheige/hephfx/hestia"
)

// consulResolverBuilder 基于 Discovery 的 gRPC resolver 构造器。
type consulResolverBuilder struct {
	discovery hestia.Discovery
	scheme    string
}

// NewConsulResolverBuilder 创建 gRPC resolver builder。
func NewConsulResolverBuilder(discovery hestia.Discovery) resolver.Builder {
	return &consulResolverBuilder{
		discovery: discovery,
		scheme:    "consul",
	}
}

// Build 实现 resolver.Builder。
func (b *consulResolverBuilder) Build(target resolver.Target,
	cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	name, version, err := parseConsulTarget(target)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &consulResolver{
		discovery: b.discovery,
		cc:        cc,
		name:      name,
		version:   version,
		cancel:    cancel,
		interval:  10 * time.Second,
	}

	services, err := r.discovery.GetServices(ctx, name, version)
	if err != nil {
		if !errors.Is(err, hestia.ErrServicesNotFound) {
			return nil, err
		}
	} else {
		r.updateState(services)
	}

	// 当 discovery 是 consulDiscovery 时复用其内部 watch 能力，否则退化为轮询。
	if cd, ok := b.discovery.(*consulDiscovery); ok {
		go r.watch(ctx, cd)
	} else {
		go r.poll(ctx)
	}

	return r, nil
}

// Scheme 实现 resolver.Builder。
func (b *consulResolverBuilder) Scheme() string {
	return b.scheme
}

// RegisterConsulResolver 使用指定 Discovery 注册 consul gRPC resolver。
func RegisterConsulResolver(discovery hestia.Discovery) {
	resolver.Register(NewConsulResolverBuilder(discovery))
}

// consulResolver 实现 resolver.Resolver
type consulResolver struct {
	discovery hestia.Discovery
	cc        resolver.ClientConn
	name      string
	version   string
	cancel    context.CancelFunc
	interval  time.Duration
}

// ResolveNow 实现 resolver.Resolver
func (r *consulResolver) ResolveNow(resolver.ResolveNowOptions) {}

// Close 实现 resolver.Resolver，停止后台 goroutine。
func (r *consulResolver) Close() {
	r.cancel()
}

func (r *consulResolver) watch(ctx context.Context, cd *consulDiscovery) {
	cd.watchWithCallback(ctx, r.name, r.version, r.updateStateWithError)
}

func (r *consulResolver) poll(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			services, err := r.discovery.GetServices(ctx, r.name, r.version)
			r.updateStateWithError(services, err)
		}
	}
}

func (r *consulResolver) updateStateWithError(services []*hestia.Service, err error) {
	if err != nil {
		if !errors.Is(err, hestia.ErrServicesNotFound) {
			r.cc.ReportError(err)
		}
		return
	}

	r.updateState(services)
}

func (r *consulResolver) updateState(services []*hestia.Service) {
	addrs := make([]resolver.Address, 0, len(services))
	for _, s := range services {
		if s.Protocol != hestia.ProtocolGRPC {
			continue
		}

		addr := resolver.Address{
			Addr:       s.Address,
			ServerName: s.Name,
		}
		addrs = append(addrs, addr)
	}

	err := r.cc.UpdateState(resolver.State{Addresses: addrs})
	if err != nil {
		log.Println("failed to update consul state err:", err)
	}
}

// parseConsulTarget 解析 consul:///service_name/version 形式的 target。
func parseConsulTarget(target resolver.Target) (name, version string, err error) {
	path := target.URL.Path
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", fmt.Errorf("consul resolver target path is empty, got: %s", target.URL.String())
	}

	if len(parts) == 1 {
		return parts[0], "", nil
	}
	return parts[0], parts[1], nil
}
