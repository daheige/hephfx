package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/daheige/hephfx/gutils"
	"github.com/daheige/hephfx/hestia"
)

var _ hestia.Registry = (*etcdRegistry)(nil)

type etcdRegistry struct {
	client   *clientv3.Client
	leaseTTL int64
	meta     *registerMeta
	prefix   string
	stop     chan struct{}
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
		stop:     make(chan struct{}, 1),
	}

	return e, nil
}

// Register service instance register
func (e *etcdRegistry) Register(s *hestia.Service) error {
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
	err = e.register(s, leaseID)
	if err != nil {
		return err
	}

	meta := &registerMeta{
		leaseID: leaseID,
	}
	err = e.keepalive(meta)
	if err != nil {
		return err
	}

	e.meta = meta
	return nil
}

func (e *etcdRegistry) register(s *hestia.Service, leaseID clientv3.LeaseID) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s/%s", e.prefix, s.Name, s.InstanceID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = e.client.Put(ctx, key, string(b), clientv3.WithLease(leaseID))
	if err != nil {
		return err
	}

	log.Printf("register service:%v instanceID:%v leaseID:%v success\n", s.Name, s.InstanceID, leaseID)
	return nil
}

// Deregister the service goes offline when the application exit
func (e *etcdRegistry) Deregister(s *hestia.Service) error {
	if s.Name == "" {
		return fmt.Errorf("missing service name in Deregister")
	}

	return e.deregister(s)
}

// String returns the name of the discovery
func (e *etcdRegistry) String() string {
	return "etcd"
}

func (e *etcdRegistry) keepalive(meta *registerMeta) error {
	keepAliveCh, err := e.client.KeepAlive(context.Background(), meta.leaseID)
	if err != nil {
		return err
	}

	go func() {
		// eat keepAliveCh channel to keep related lease alive.
		log.Printf("start keepalive leaseID:%v for etcd registry", meta.leaseID)
		for range keepAliveCh {
			select {
			case <-e.stop:
				log.Printf("stop keepalive leaseID:%v for etcd registry", meta.leaseID)
				return
			default:
				// log.Printf("keepalive leaseID:%v for etcd registry\n", meta.leaseID)
			}
		}
	}()

	return nil
}

func (e *etcdRegistry) deregister(s *hestia.Service) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	// validate address
	_, err := hestia.Resolve(s.Address)
	if err != nil {
		return fmt.Errorf("failed to resolve address:%v error:%v", s.Address, err)
	}

	key := fmt.Sprintf("%s/%s/%s", e.prefix, s.Name, s.InstanceID)
	_, err = e.client.Delete(ctx, key)
	if err != nil {
		return err
	}

	close(e.stop)
	return nil
}
