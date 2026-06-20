package hestia

import "context"

// Registry is extension interface of service registry
type Registry interface {
	// Register service instance register
	Register(ctx context.Context, s *Service) error

	// Deregister the service goes offline when the application exit
	Deregister(ctx context.Context, s *Service) error

	// String returns the name of the registry
	String() string
}
