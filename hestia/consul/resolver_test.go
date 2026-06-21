package consul

import (
	"net/url"
	"testing"

	"google.golang.org/grpc/resolver"

	"github.com/daheige/hephfx/hestia"
)

func parseURL(raw string) url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return *u
}

func TestParseConsulTarget(t *testing.T) {
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
				URL: parseURL("consul:///order_service/v1"),
			},
			wantSvc: "order_service",
			wantVer: "v1",
		},
		{
			name: "name only",
			target: resolver.Target{
				URL: parseURL("consul:///order_service"),
			},
			wantSvc: "order_service",
			wantVer: "",
		},
		{
			name: "empty path",
			target: resolver.Target{
				URL: parseURL("consul:///"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ver, err := parseConsulTarget(tt.target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseConsulTarget() error = %v, wantErr %v", err, tt.wantErr)
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

func TestBuildVersionFilter(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"empty version", "", ""},
		{"v1", "v1", "version:v1"},
		{"v2", "v2", "version:v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildVersionFilter(tt.version)
			if got != tt.want {
				t.Fatalf("buildVersionFilter(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

func TestBuildTags(t *testing.T) {
	s := &hestia.Service{
		Name:       "order",
		Version:    "v1",
		InstanceID: "uuid-1",
		Protocol:   hestia.ProtocolGRPC,
	}
	tags := buildTags(s, "hestia")

	hasPrefix := false
	hasVersion := false
	hasProtocol := false
	hasInstanceID := false
	for _, tag := range tags {
		switch tag {
		case "prefix:hestia":
			hasPrefix = true
		case "version:v1":
			hasVersion = true
		case "protocol:GRPC":
			hasProtocol = true
		case "instance_id:uuid-1":
			hasInstanceID = true
		}
	}

	if !hasPrefix {
		t.Error("missing prefix tag")
	}
	if !hasVersion {
		t.Error("missing version tag")
	}
	if !hasProtocol {
		t.Error("missing protocol tag")
	}
	if !hasInstanceID {
		t.Error("missing instance_id tag")
	}
}

func TestTagValue(t *testing.T) {
	tags := []string{"version:v1", "protocol:GRPC", "instance_id:uuid-1"}

	if v := tagValue(tags, "version:"); v != "v1" {
		t.Errorf("version = %q, want v1", v)
	}
	if v := tagValue(tags, "protocol:"); v != "GRPC" {
		t.Errorf("protocol = %q, want GRPC", v)
	}
	if v := tagValue(tags, "missing:"); v != "" {
		t.Errorf("missing = %q, want empty", v)
	}
}

func TestNormalizePrefix(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"/hestia/registry-consul", "hestia/registry-consul"},
		{"hestia", "hestia"},
		{"/hestia/", "hestia"},
		{"///hestia///", "hestia"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := normalizePrefix(tt.in); got != tt.want {
			t.Errorf("normalizePrefix(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
