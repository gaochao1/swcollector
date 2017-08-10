package g

import (
	"encoding/json"
	"log"

	"sync"

	"github.com/toolkits/file"
)

type MetricConfig struct {
	IpRange []string `json:"ipRange"`
	Metric  string   `json:metric`
	Tag     string   `json:tag`
	Type    string   `json:type`
	Oid     string   `json:oid`
}
type CustomConfig struct {
	Metrics []*MetricConfig `json:"metrics`
}

var (
	CustConfigFile string
	custconfig     *CustomConfig
	custlock       = new(sync.RWMutex)
)

func CustConfig() *CustomConfig {
	custlock.RLock()
	defer custlock.RUnlock()
	return custconfig
}

func ParseCustConfig(cfg string) {

	if !file.IsExist(cfg) {
		log.Fatalln("config file:", cfg, "is not existent")
	}
	CustConfigFile = cfg
	configContent, err := file.ToTrimString(cfg)
	if err != nil {
		log.Fatalln("read config file:", cfg, "fail:", err)
	}
	var c CustomConfig
	err = json.Unmarshal([]byte(configContent), &c)
	if err != nil {
		log.Fatalln("parse config file:", cfg, "fail:", err)
	}
	custlock.Lock()
	defer custlock.Unlock()
	custconfig = &c

	log.Println("read customconfig file:", cfg, "successfully")

}
