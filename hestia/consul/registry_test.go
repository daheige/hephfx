package consul

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/daheige/hephfx/hestia"
)

func TestRegistry(t *testing.T) {
	r, err := NewRegistry([]string{
		"127.0.0.1:8500",
	})
	if err != nil {
		t.Fatal(err)
	}

	s := &hestia.Service{
		Network: "tcp",
		Name:    "my-test",
		Address: "127.0.0.1:8080",
		Version: "v1",
		Created: time.Now().Format("2006-01-02 15:04:05"),
	}

	ctx := context.Background()
	err = r.Register(ctx, s)
	if err != nil {
		log.Printf("failed to register service: %v", err)
	}

	time.Sleep(100 * time.Second)

	// mock service exit
	err = r.Deregister(ctx, s)
	if err != nil {
		log.Printf("failed to deregister service: %v", err)
	}
}

func TestDiscovery(t *testing.T) {
	d, err := NewDiscovery([]string{
		"127.0.0.1:8500",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	services, err := d.GetServices(ctx, "my-test", "v1")
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("services count: %d", len(services))
	for _, svc := range services {
		log.Printf("service: %s, address: %s, version: %s", svc.Name, svc.Address, svc.Version)
	}

	// test with watch
	d2, err := NewDiscovery([]string{
		"127.0.0.1:8500",
	}, WithEnableWatched())
	if err != nil {
		t.Fatal(err)
	}

	services2, err := d2.GetServices(ctx, "my-test", "v1")
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("watch services count: %d", len(services2))
	for _, svc := range services2 {
		log.Printf("watch service: %s, address: %s, version: %s", svc.Name, svc.Address, svc.Version)
	}

	time.Sleep(60 * time.Second)
}
