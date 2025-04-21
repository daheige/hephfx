package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func newEtcdClient(opt *Options) (*clientv3.Client, error) {
	config := clientv3.Config{
		Endpoints:   opt.endpoints,
		DialTimeout: opt.dialTimeout,
		Username:    opt.username,
		Password:    opt.password,
	}

	client, err := clientv3.New(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func grantLease(etcdClient *clientv3.Client, leaseTTL int64) (clientv3.LeaseID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	resp, err := etcdClient.Grant(ctx, leaseTTL)
	if err != nil {
		return clientv3.NoLease, err
	}

	return resp.ID, nil
}
