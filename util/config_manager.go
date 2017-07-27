package util

import (
	"barrage/logger"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

type ConfigManager struct {
	srvConf ServicesConfig
	conf    string
}

type IOServiceConf struct {
	Srvworker int
	Srvqueue  int

	Ioworker int
	Ioqueue  int

	Matrixbucket int
	Matrixsize   int
}

var (
	cm   *ConfigManager
	once sync.Once
)

func NewConfigManager(config string) *ConfigManager {
	once.Do(func() {
		cm = &ConfigManager{
			conf: config,
		}
		var conf ServicesConfig
		conf_path, _ := filepath.Abs(config)
		if _, err := toml.DecodeFile(conf_path, &conf); err != nil {
			logger.Error("DecodeFile error %v", err)
			return
		}
		cm.srvConf = conf
	})
	return cm
}

type SrvInfo map[string]serviceInfo

type HttpConfig struct {
	Localaddr  string
	Remoteaddr string
}

type ZkConfig struct {
	Addrs   []string
	Timeout Duration
	SrvId   string
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type ServicesConfig struct {
	Services SrvInfo
	Engine   IOServiceConf
	Http     HttpConfig
	Zk       ZkConfig
}

type serviceInfo struct {
	Addr string
	Name string
}

func GetServicesConfig() SrvInfo {
	return cm.srvConf.Services
}

func GetIOServiceConf() IOServiceConf {
	return cm.srvConf.Engine
}

func GetHttpConfig() HttpConfig {
	return cm.srvConf.Http
}

func GetZkConfig() ZkConfig {
	return cm.srvConf.Zk
}

func GetConf() string {
	return cm.conf
}

func GetManager() *ConfigManager {
	return cm
}
