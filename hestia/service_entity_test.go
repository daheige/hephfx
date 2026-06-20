package hestia

import (
	"testing"
)

func TestRoundRobinHandler(t *testing.T) {
	services := []*Service{
		{Address: "a:1"},
		{Address: "a:2"},
		{Address: "a:3"},
	}

	seen := make(map[string]int)
	for i := 0; i < len(services)*3; i++ {
		svc := RoundRobinHandler(services)
		if svc == nil {
			t.Fatal("got nil service")
		}
		seen[svc.Address]++
	}

	for _, svc := range services {
		if seen[svc.Address] != 3 {
			t.Fatalf("expected 3 selections for %s, got %d", svc.Address, seen[svc.Address])
		}
	}
}

func TestRoundRobinHandlerEmpty(t *testing.T) {
	if RoundRobinHandler(nil) != nil {
		t.Fatal("expected nil for empty services")
	}
}
