package g

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/toolkits/file"
)

type SwitchConfig struct {
	Enabled  bool     `json:"enabled"`
	IpRange  []string `json:"ipRange"`
	Interval int      `json:"interval"`

	PingTimeout int `json:"pingTimeout"`
	PingRetry   int `json:"pingRetry"`

	Community   string `json:"community"`
	SnmpTimeout int    `json:"snmpTimeout"`
	SnmpRetry   int    `json:"snmpRetry"`

	IgnoreIface []string `json:"ignoreIface"`
	IgnorePkt   bool     `json:"ignorePkt"`
	LimitConcur int      `json:"limitConcur"`
	HightIface  []string `json:"hightIface"`
}

type HeartbeatConfig struct {
	Enabled  bool   `json:"enabled"`
	Addr     string `json:"addr"`
	Interval int    `json:"interval"`
	Timeout  int    `json:"timeout"`
}

type TransferConfig struct {
	Enabled  bool   `json:"enabled"`
	Addr     string `json:"addr"`
	Interval int    `json:"interval"`
	Timeout  int    `json:"timeout"`
}

type HttpConfig struct {
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
}

type CollectorConfig struct {
	IfacePrefix []string `json:"ifacePrefix"`
	LanIpnet    []string `json:"lanipnet"`
}

type GlobalConfig struct {
	Debug         bool             `json:"debug"`
	Rate          bool             `json:"rate"`
	IP            string           `json:"ip"`
	Hostname      string           `json:"hostname"`
	Switch        *SwitchConfig    `json:"switch"`
	Heartbeat     *HeartbeatConfig `json:"heartbeat"`
	Transfer      *TransferConfig  `json:"transfer"`
	Http          *HttpConfig      `json:"http"`
	Collector     *CollectorConfig `json:"collector"`
	IgnoreMetrics map[string]bool  `json:"ignore"`
}

var (
	ConfigFile string
	config     *GlobalConfig
	lock       = new(sync.RWMutex)
)

func Config() *GlobalConfig {
	lock.RLock()
	defer lock.RUnlock()
	return config
}

func Hostname() (string, error) {
	hostname := Config().Hostname
	if hostname != "" {
		return hostname, nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Println("ERROR: os.Hostname() fail", err)
	}
	return hostname, err
}

func IP() string {
	ip := Config().IP
	if ip != "" {
		// use ip in configuration
		return ip
	}

	if len(LocalInterNetIps) > 0 {
		ip = LocalInterNetIps[0]
	}

	return ip
}

func ConfigIp() (string, bool) {
	ip := Config().IP
	if ip != "" {
		// use ip in configuration
		return ip, true
	} else {
		return "", false
	}
}

func ParseConfig(cfg string) {
	if cfg == "" {
		log.Fatalln("use -c to specify configuration file")
	}

	if !file.IsExist(cfg) {
		log.Fatalln("config file:", cfg, "is not existent. maybe you need `mv cfg.example.json cfg.json`")
	}

	ConfigFile = cfg

	configContent, err := file.ToTrimString(cfg)
	if err != nil {
		log.Fatalln("read config file:", cfg, "fail:", err)
	}

	var c GlobalConfig
	err = json.Unmarshal([]byte(configContent), &c)
	if err != nil {
		log.Fatalln("parse config file:", cfg, "fail:", err)
	}

	lock.Lock()
	defer lock.Unlock()

	config = &c

	log.Println("read config file:", cfg, "successfully")

}
