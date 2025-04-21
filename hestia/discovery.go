package hestia

// Discovery service discovery interface
type Discovery interface {
	// GetServices returns a list of instances
	// After we obtain the service instance, we can get the currently available service instance
	// from the service list according to different strategies.
	GetServices(name string) ([]*Service, error)

	// Get returns an available service instance based on the specified service selection strategy.
	// the selection strategy is RoundRobinHandler
	Get(name string, strategyHandler ...StrategyHandler) (*Service, error)

	// String returns the name of the resolver.
	String() string
}
