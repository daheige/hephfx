package bridge

import (
	"fmt"
	"strings"

	"github.com/daheige/hephfx/settings"
)

// Config bridge 客户端配置。
type Config struct {
	Services []ServiceConfig `mapstructure:"services"`
}

// ServiceConfig 下游 gRPC 微服务配置。
type ServiceConfig struct {
	Name     string            `mapstructure:"name"`     // 逻辑服务名，如 uc-svc
	Target   string            `mapstructure:"target"`   // 下游 gRPC 地址，如 uc.cluster.local:8080
	Service  string            `mapstructure:"service"`  // gRPC 完整服务名，如 Hello.Greeter；为空时使用 Name
	Version  string            `mapstructure:"version"`  // 协议版本号，预留字段
	Metadata map[string]string `mapstructure:"metadata"` // 该服务默认透传元数据
}

// LoadConfig 从 path 加载 app.yaml 配置文件。
func LoadConfig(path string) (*Config, error) {
	cfg, err := settings.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return configFromSettings(cfg)
}

// LoadConfigFrom 从已加载的 settings.Config 构造 Bridge 配置。
func LoadConfigFrom(cfg settings.Config) (*Config, error) {
	return configFromSettings(cfg)
}

func configFromSettings(s settings.Config) (*Config, error) {
	var cfg Config
	if s.IsSet("bridge_services") {
		if err := s.ReadSection("bridge_services", &cfg.Services); err != nil {
			return nil, fmt.Errorf("read services section: %w", err)
		}
	}

	return &cfg, nil
}

func (c ServiceConfig) fullServiceName() string {
	name := strings.Trim(c.Service, "/")
	return name
}
