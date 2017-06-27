package g

import (
	"encoding/json"
	"log"

	"sync"

	"github.com/toolkits/file"
)

type HostsConfig struct {
	Hosts map[string]string `json:"hosts"`
}

var (
	HostConfigFile string
	hostconfig     *HostsConfig
	hostlock       = new(sync.RWMutex)
)

func HostConfig() *HostsConfig {
	hostlock.RLock()
	defer hostlock.RUnlock()
	return hostconfig
}

func ParseHostConfig(cfg string) {
	if !file.IsExist(cfg) {
		log.Fatalln("config file:", cfg, "is not existent")
	}

	HostConfigFile = cfg

	configContent, err := file.ToTrimString(cfg)
	if err != nil {
		log.Fatalln("read config file:", cfg, "fail:", err)
	}

	var c HostsConfig
	err = json.Unmarshal([]byte(configContent), &c)
	if err != nil {
		log.Fatalln("parse config file:", cfg, "fail:", err)
	}

	hostlock.Lock()
	defer hostlock.Unlock()

	hostconfig = &c

	log.Println("read hostconfig file:", cfg, "successfully")

}
