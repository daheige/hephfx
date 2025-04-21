package etcd

import (
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
		Network:    "tcp",
		Name:       "my-test",
		Address:    "127.0.0.1:12379",
		InstanceID: "1234",
		Version:    "v1",
		Created:    time.Now().Format("2006-01-02 15:04:05"),
	}

	err = r.Register(s)
	if err != nil {
		log.Printf("failed to register service: %v", err)
	}

	time.Sleep(30 * time.Second)

	// mock service exit
	err = r.Deregister(s)
	if err != nil {
		log.Printf("failed to register service: %v", err)
	}
}
