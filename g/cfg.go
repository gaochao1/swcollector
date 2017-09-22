package g

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/toolkits/file"
)

type DebugmetricConfig struct {
	Endpoints []string `json:"endpoints`
	Metrics   []string `json:"metrics`
	Tags      string   `json:"tags"`
}

type SwitchConfig struct {
	Enabled bool     `json:"enabled"`
	IpRange []string `json:"ipRange"`
	Gosnmp  bool     `json:"gosnmp"`

	PingTimeout int `json:"pingTimeout"`
	PingRetry   int `json:"pingRetry"`

	Community   string `json:"community"`
	SnmpTimeout int    `json:"snmpTimeout"`
	SnmpRetry   int    `json:"snmpRetry"`

	IgnoreIface           []string `json:"ignoreIface"`
	IgnoreOperStatus      bool     `json:"ignoreOperStatus"`
	Speedlimit            float64  `json:"speedlimit"`
	IgnorePkt             bool     `json:"ignorePkt"`
	Pktlimit              float64  `json:"pktlimit"`
	IgnoreBroadcastPkt    bool     `json:"ignoreBroadcastPkt"`
	BroadcastPktlimit     float64  `josn:"broadcastPktlimit"`
	IgnoreMulticastPkt    bool     `json:"ignoreMulticastPkt"`
	MulticastPktlimit     float64  `json:"multicastPktlimit"`
	IgnoreDiscards        bool     `json:"ignoreDiscards"`
	DiscardsPktlimit      float64  `json:"discardsPktlimit"`
	IgnoreErrors          bool     `json:"ignoreErrors"`
	ErrorsPktlimit        float64  `json:"errorsPktlimit"`
	IgnoreUnknownProtos   bool     `json:"ignoreUnknownProtos`
	UnknownProtosPktlimit float64  `json:"unknownProtosPktlimit"`
	IgnoreOutQLen         bool     `json:"ignoreOutQLen`
	OutQLenPktlimit       float64  `json:"outQLenPktlimit"`
	LimitCon              int      `json:limitCon`
	LimitConcur           int      `json:"limitConcur"`
	FastPingMode          bool     `json:"fastPingMode"`
}

type TransferConfig struct {
	Enabled  bool   `json:"enabled"`
	Addr     string `json:"addr"`
	Interval int    `json:"interval"`
	Timeout  int    `json:"timeout"`
}

type HttpConfig struct {
	Enabled  bool     `json:"enabled"`
	Listen   string   `json:"listen"`
	TrustIps []string `json:trustIps`
}

type SwitchHostsConfig struct {
	Enabled bool   `json:enabled`
	Hosts   string `json:hosts`
}

type CustomMetricsConfig struct {
	Enabled  bool   `json:enbaled`
	Template string `json:template`
}

type GlobalConfig struct {
	Debug         bool                 `json:"debug"`
	Debugmetric   *DebugmetricConfig   `json:"debugmetric`
	Switch        *SwitchConfig        `json:"switch"`
	Transfer      *TransferConfig      `json:"transfer"`
	SwitchHosts   *SwitchHostsConfig   `json:switchhosts`
	CustomMetrics *CustomMetricsConfig `json:customMetrics`
	Http          *HttpConfig          `json:"http"`
}

var (
	ConfigFile string
	config     *GlobalConfig
	reloadType bool
	lock       = new(sync.RWMutex)
	rlock      = new(sync.RWMutex)
)

func SetReloadType(t bool) {
	rlock.RLock()
	defer rlock.RUnlock()
	reloadType = t
	return
}

func ReloadType() bool {
	rlock.RLock()
	defer rlock.RUnlock()
	return reloadType
}

func Config() *GlobalConfig {
	lock.RLock()
	defer lock.RUnlock()
	return config
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
	SetReloadType(false)
	log.Println("read config file:", cfg, "successfully")

}
