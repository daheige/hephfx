package bridge

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cfg := &Config{
		Services: []ServiceConfig{
			{Name: "uc-svc", Target: "uc.cluster.local:8080", Service: "user.UserService"},
			{Name: "rbac-svc", Target: "rbac.cluster.local:8080", Service: "rbac.RBAC"},
		},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer client.Close()

	svc, err := client.Service("rbac-svc")
	if err != nil {
		t.Fatalf("get service failed: %v", err)
	}
	if svc.Name() != "rbac-svc" {
		t.Fatalf("service name mismatch: got %s", svc.Name())
	}
	if svc.FullServiceName() != "rbac.RBAC" {
		t.Fatalf("full service name mismatch: got %s", svc.FullServiceName())
	}
}

func TestServiceNotFound(t *testing.T) {
	cfg := &Config{
		Services: []ServiceConfig{
			{Name: "uc-svc", Target: "uc.cluster.local:8080"},
		},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer client.Close()

	_, err = client.Service("not-exist")
	if err == nil {
		t.Fatalf("expected error for not exist service")
	}
}

func TestFullServiceName(t *testing.T) {
	cases := []struct {
		cfg  ServiceConfig
		want string
	}{
		{cfg: ServiceConfig{Name: "uc-svc", Service: "uc.UserService"}, want: "uc.UserService"},
		{cfg: ServiceConfig{Name: "greeter-svc", Service: "Hello.Greeter"}, want: "Hello.Greeter"},
	}

	for _, c := range cases {
		got := c.cfg.fullServiceName()
		if got != c.want {
			t.Fatalf("fullServiceName(%+v) = %s, want %s", c.cfg, got, c.want)
		}
	}
}
