package etcd

import (
	"net/url"
	"testing"

	"google.golang.org/grpc/resolver"
)

func parseURL(raw string) url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return *u
}

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  resolver.Target
		wantSvc string
		wantVer string
		wantErr bool
	}{
		{
			name: "name and version",
			target: resolver.Target{
				URL: parseURL("etcd:///order_service/v1"),
			},
			wantSvc: "order_service",
			wantVer: "v1",
		},
		{
			name: "name only",
			target: resolver.Target{
				URL: parseURL("etcd:///order_service"),
			},
			wantSvc: "order_service",
			wantVer: "",
		},
		{
			name: "empty path",
			target: resolver.Target{
				URL: parseURL("etcd:///"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ver, err := parseTarget(tt.target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if svc != tt.wantSvc {
				t.Fatalf("got service %q, want %q", svc, tt.wantSvc)
			}
			if ver != tt.wantVer {
				t.Fatalf("got version %q, want %q", ver, tt.wantVer)
			}
		})
	}
}
