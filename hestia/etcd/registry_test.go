package etcd

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	"github.com/daheige/hephfx/hestia"
)

func TestRegistry(t *testing.T) {
	r, err := NewRegistry([]string{
		"http://127.0.0.1:12379",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := &hestia.Service{
		Network: "tcp",
		Name:    "my-test",
		Address: "127.0.0.1:12379",
		Version: "v1",
		Created: time.Now().Format("2006-01-02 15:04:05"),
	}

	err = r.Register(s)
	if err != nil {
		log.Printf("failed to register service: %v", err)
	}

	time.Sleep(100 * time.Second)

	// mock service exit
	err = r.Deregister(s)
	if err != nil {
		log.Printf("failed to register service: %v", err)
	}
}

func TestDiscovery(t *testing.T) {
	r, err := NewDiscovery([]string{
		"http://127.0.0.1:12379",
	})

	// 测试watch功能
	// r, err := NewDiscovery([]string{
	// 	"http://127.0.0.1:12379",
	// }, WithDiscoveryWatched())

	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ { // mock obtaining the service list multiple times
		services, err := r.GetServices("my-test")
		if err != nil {
			t.Fatal(err)
		}

		b, _ := json.Marshal(services)
		log.Printf("services: %v", string(b))
	}

	time.Sleep(2 * time.Second)
	services, err := r.GetServices("my-test")
	if err != nil {
		t.Fatal(err)
	}

	b, _ := json.Marshal(services)
	log.Printf("services: %v", string(b))

	time.Sleep(60 * time.Second)
}
