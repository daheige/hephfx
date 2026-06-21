package consul

import "time"

// Options consul options
type Options struct {
	endpoints                      []string      // consul agent地址列表
	dialTimeout                    time.Duration // 默认5s
	ttl                            string        // health check ttl, 默认30s
	deregisterCriticalServiceAfter string        // 服务被标记critical后自动注销的时间，默认90s
	prefix                         string        // 服务名前缀，default:hestia
	token                          string        // consul ACL token
	validateAddress                bool          // 是否校验address有效性，default:false
	disableWatch                   bool          // 是否禁用watch，default:true
}

// Option consul functional option
type Option func(*Options)

// WithDialTimeout set dial timeout
func WithDialTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.dialTimeout = d
	}
}

// WithTTL set health check ttl
func WithTTL(ttl string) Option {
	return func(o *Options) {
		o.ttl = ttl
	}
}

// WithDeregisterCriticalServiceAfter set the time after which a critical service
// is automatically deregistered. Should be longer than ttl.
func WithDeregisterCriticalServiceAfter(d string) Option {
	return func(o *Options) {
		o.deregisterCriticalServiceAfter = d
	}
}

// WithPrefix set service name prefix
func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.prefix = prefix
	}
}

// WithToken set consul ACL token
func WithToken(token string) Option {
	return func(o *Options) {
		o.token = token
	}
}

// WithValidateAddress whether to validate address
func WithValidateAddress(validate bool) Option {
	return func(o *Options) {
		o.validateAddress = validate
	}
}

// WithEnableWatched enable consul watch (blocking query)
func WithEnableWatched() Option {
	return func(o *Options) {
		o.disableWatch = false
	}
}
