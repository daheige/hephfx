package settings

import (
	"encoding/json"
	"log"
	"testing"
)

// AppConfig config struct
type AppConfig struct {
	AppEnv   string `mapstructure:"app_env" json:"app_env"`
	AppDebug string `mapstructure:"app_debug" json:"app_debug"`
	AppName  string `mapstructure:"app_name" json:"app_name"`
	AppPort  uint16 `mapstructure:"app_port" json:"app_port"`
}

/*
=== RUN   TestNew
2025/04/20 10:57:49 &{local 1 hephfx 8090}
--- PASS: TestNew (0.00s)
PASS
*/
func TestNew(t *testing.T) {
	path := "./app.test.yaml"
	conf := New(WithConfigFile(path))
	err := conf.Load()
	if err != nil {
		t.Fatal(err)
	}

	appConfig := &AppConfig{}
	err = conf.ReadSection("app_config", appConfig)
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("%v", appConfig)
}

/*
=== RUN   TestLoad
2025/04/20 10:57:31 config filename: ./app.test.yaml  dir: .
2025/04/20 10:57:31 {"app_env":"local","app_debug":"1","app_name":"hephfx","app_port":8090}
--- PASS: TestLoad (0.00s)
PASS
*/
func TestLoad(t *testing.T) {
	path := "./app.test.yaml"
	conf, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	appConfig := &AppConfig{}
	err = conf.ReadSection("app_config", appConfig)
	if err != nil {
		t.Fatal(err)
	}

	b, _ := json.Marshal(appConfig)
	log.Printf("%s", string(b))
}
