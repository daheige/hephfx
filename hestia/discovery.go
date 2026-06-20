package hestia

import "context"

// Discovery service discovery interface
type Discovery interface {
	// GetServices returns a list of instances
	// After we obtain the service instance, we can get the currently available service instance
	// from the service list according to different strategies.
	// version 版本号参数，如果传递为空，就没有版本区分
	GetServices(ctx context.Context, name string, version string) ([]*Service, error)

	// Get returns an available service instance based on the specified service selection strategy.
	// the selection strategy is RoundRobinHandler
	Get(ctx context.Context, name string, version string, strategyHandler ...StrategyHandler) (*Service, error)

	// String returns the name of the resolver.
	String() string
}
