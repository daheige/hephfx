package etcd

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

// etcdResolverBuilder 基于 Discovery 的 gRPC resolver 构造器。
type etcdResolverBuilder struct {
	discovery hestia.Discovery
	scheme    string
}

// NewEtcdResolverBuilder 创建 gRPC resolver builder。
// 参数 discovery 用于服务发现
// scheme使用etcd
func NewEtcdResolverBuilder(discovery hestia.Discovery) resolver.Builder {
	return &etcdResolverBuilder{
		discovery: discovery,
		scheme:    "etcd",
	}
}

// Build 实现 resolver.Builder。
func (b *etcdResolverBuilder) Build(target resolver.Target,
	cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	name, version, err := parseTarget(target)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &etcdResolver{
		discovery: b.discovery,
		cc:        cc,
		name:      name,
		version:   version,
		cancel:    cancel,
		interval:  10 * time.Second,
	}

	services, err := r.discovery.GetServices(ctx, name, version)
	if err != nil {
		return nil, err
	}

	r.updateState(services)

	// 当 discovery 是 etcdDiscovery 时复用其内部 watch 能力，否则退化为轮询。
	if ed, ok := b.discovery.(*etcdDiscovery); ok {
		go r.watch(ctx, ed)
	} else {
		go r.poll(ctx)
	}

	return r, nil
}

// Scheme 实现 resolver.Builder。
func (b *etcdResolverBuilder) Scheme() string {
	return b.scheme
}

// RegisterEtcdResolver 使用指定 Discovery 注册 etcd gRPC resolver。
func RegisterEtcdResolver(discovery hestia.Discovery) {
	resolver.Register(NewEtcdResolverBuilder(discovery))
}

// etcdResolver 实现 resolver.Resolver
type etcdResolver struct {
	discovery hestia.Discovery
	cc        resolver.ClientConn
	name      string
	version   string
	cancel    context.CancelFunc
	interval  time.Duration
}

// ResolveNow 实现 resolver.Resolver，目前不需要主动触发。
func (r *etcdResolver) ResolveNow(resolver.ResolveNowOptions) {}

// Close 实现 resolver.Resolver，停止后台 goroutine。
func (r *etcdResolver) Close() {
	r.cancel()
}

func (r *etcdResolver) watch(ctx context.Context, ed *etcdDiscovery) {
	ed.watchWithCallback(ctx, r.name, r.version, r.updateStateWithError)
}

func (r *etcdResolver) poll(ctx context.Context) {
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

func (r *etcdResolver) updateStateWithError(services []*hestia.Service, err error) {
	if err != nil {
		if !errors.Is(err, hestia.ErrServicesNotFound) {
			r.cc.ReportError(err)
		}
		return
	}

	r.updateState(services)
}

func (r *etcdResolver) updateState(services []*hestia.Service) {
	addrs := make([]resolver.Address, 0, len(services))
	for _, s := range services {
		addr := resolver.Address{
			Addr:       s.Address,
			ServerName: s.Name,
		}
		addrs = append(addrs, addr)
	}

	err := r.cc.UpdateState(resolver.State{Addresses: addrs})
	if err != nil {
		log.Println("failed to update etcd state err:", err)
	}
}

// parseTarget 解析 etcd:///service_name/version 形式的 target。
func parseTarget(target resolver.Target) (name, version string, err error) {
	path := target.URL.Path
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", fmt.Errorf("etcd resolver target path is empty, got: %s", target.URL.String())
	}
	if len(parts) == 1 {
		return parts[0], "", nil
	}
	return parts[0], parts[1], nil
}
