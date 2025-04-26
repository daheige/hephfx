package hestia

// Registry is extension interface of service registry
type Registry interface {
	// Register service instance register
	Register(s *Service) error

	// Deregister the service goes offline when the application exit
	Deregister(s *Service) error

	// String returns the name of the registry
	String() string
}
