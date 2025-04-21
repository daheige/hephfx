package etcd

import (
	"time"
)

// Options etcd options
type Options struct {
	endpoints   []string      // etcd节点列表
	dialTimeout time.Duration // 默认5s
	leaseTTL    int64         // etcd lease时间，单位s，默认60s
	prefix      string        // 默认:hestia/registry-etcd

	// etcd 用户名和密码可选
	username string
	password string
}

// Option etcdImpl functional option
type Option func(*Options)

// WithDialTimeout set dialTimeout
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(o *Options) {
		o.dialTimeout = dialTimeout
	}
}

// WithUsername 设置 username
func WithUsername(username string) Option {
	return func(o *Options) {
		o.username = username
	}
}

// WithPassword 设置 password
func WithPassword(password string) Option {
	return func(o *Options) {
		o.password = password
	}
}

// WithPrefix set etcd prefix
func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.prefix = prefix
	}
}

// WithLeaseTTL set etcd lease ttl
func WithLeaseTTL(ttl int64) Option {
	return func(o *Options) {
		o.leaseTTL = ttl
	}
}
