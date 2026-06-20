package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/daheige/hephfx/gutils"
	"github.com/daheige/hephfx/hestia"
)

var _ hestia.Registry = (*etcdRegistry)(nil)

type etcdRegistry struct {
	client          *clientv3.Client
	leaseTTL        int64
	meta            *registerMeta
	prefix          string
	keepaliveCtx    context.Context
	keepaliveCancel context.CancelFunc
}

type registerMeta struct {
	leaseID clientv3.LeaseID
}

// NewRegistry create a registry interface instance
func NewRegistry(endpoints []string, opts ...Option) (hestia.Registry, error) {
	opt := &Options{
		endpoints:   endpoints,
		dialTimeout: 5 * time.Second,
		prefix:      "hestia/registry-etcd",
		leaseTTL:    60,
	}

	for _, o := range opts {
		o(opt)
	}

	client, err := newEtcdClient(opt)
	if err != nil {
		return nil, err
	}

	e := &etcdRegistry{
		client:   client,
		leaseTTL: opt.leaseTTL,
		prefix:   opt.prefix,
	}

	e.prefix = strings.TrimPrefix(e.prefix, "/")
	e.prefix = strings.TrimSuffix(e.prefix, "/")
	e.prefix = fmt.Sprintf("/%s", e.prefix) // 格式为/services

	return e, nil
}

// Register service instance register
func (e *etcdRegistry) Register(ctx context.Context, s *hestia.Service) error {
	if s.InstanceID == "" {
		s.InstanceID = gutils.Uuid()
	}

	// validate address
	address, err := hestia.Resolve(s.Address)
	if err != nil {
		return fmt.Errorf("failed to resolve address:%v error:%v", s.Address, err)
	}

	s.Address = address
	leaseID, err := grantLease(e.client, e.leaseTTL)
	if err != nil {
		return err
	}

	// register service
	err = e.register(ctx, s, leaseID)
	if err != nil {
		return err
	}

	meta := &registerMeta{
		leaseID: leaseID,
	}

	if e.keepaliveCancel != nil {
		e.keepaliveCancel()
	}
	e.keepaliveCtx, e.keepaliveCancel = context.WithCancel(context.Background())
	err = e.keepalive(e.keepaliveCtx, meta)
	if err != nil {
		return err
	}

	e.meta = meta
	return nil
}

func (e *etcdRegistry) register(ctx context.Context, s *hestia.Service, leaseID clientv3.LeaseID) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	key := e.registerKey(s)
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	_, err = e.client.Put(ctx, key, string(b), clientv3.WithLease(leaseID))
	if err != nil {
		return err
	}

	log.Printf("register service:%v version:%s instanceID:%v leaseID:%v success\n",
		s.Name, s.Version, s.InstanceID, leaseID)
	return nil
}

// Deregister the service goes offline when the application exit
func (e *etcdRegistry) Deregister(ctx context.Context, s *hestia.Service) error {
	if s.Name == "" {
		return errors.New("missing service name in Deregister")
	}

	return e.deregister(ctx, s)
}

// String returns the name of the discovery
func (e *etcdRegistry) String() string {
	return "etcd"
}

func (e *etcdRegistry) keepalive(ctx context.Context, meta *registerMeta) error {
	keepAliveCh, err := e.client.KeepAlive(ctx, meta.leaseID)
	if err != nil {
		return err
	}

	go func() {
		log.Printf("start keepalive leaseID:%v for etcd registry", meta.leaseID)
		for {
			select {
			case <-ctx.Done():
				log.Printf("stop keepalive leaseID:%v for etcd registry: %v", meta.leaseID, ctx.Err())
				return
			case _, ok := <-keepAliveCh:
				if !ok {
					log.Printf("keepalive channel closed leaseID:%v", meta.leaseID)
					return
				}
			}
		}
	}()

	return nil
}

func (e *etcdRegistry) registerKey(s *hestia.Service) string {
	var key string
	if s.Version != "" {
		key = fmt.Sprintf("%s/%s/%s/%s", e.prefix, s.Name, s.Version, s.InstanceID)
	} else {
		key = fmt.Sprintf("%s/%s/%s", e.prefix, s.Name, s.InstanceID)
	}

	return key
}

func (e *etcdRegistry) deregister(ctx context.Context, s *hestia.Service) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	// validate address
	_, err := hestia.Resolve(s.Address)
	if err != nil {
		return fmt.Errorf("failed to resolve address:%v error:%v", s.Address, err)
	}

	key := e.registerKey(s)
	_, err = e.client.Delete(ctx, key)
	if err != nil {
		return err
	}

	log.Printf("deregister service:%v version:%s instanceID:%v success\n", s.Name, s.Version, s.InstanceID)

	if e.keepaliveCancel != nil {
		e.keepaliveCancel()
	}
	return nil
}
