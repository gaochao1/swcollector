package g

import (
	"log"
	"os"
	"strings"

	"time"

	"github.com/open-falcon/common/model"
	"github.com/toolkits/net"
)

var Root string

func InitRootDir() {
	var err error
	Root, err = os.Getwd()
	if err != nil {
		log.Fatalln("getwd fail:", err)
	}
}

var LocalIps []string

func InitLocalIps() {
	var err error
	LocalIps, err = net.IntranetIP()
	if err != nil {
		log.Fatalln("get intranet ip fail:", err)
	}
}

var (
	TransferClient *SingleConnRpcClient
)

func InitRpcClients() {
	if Config().Transfer.Enabled {
		TransferClient = &SingleConnRpcClient{
			RpcServer: Config().Transfer.Addr,
			Timeout:   time.Duration(Config().Transfer.Timeout) * time.Millisecond,
		}
	}
}

func SendToTransfer(metrics []*model.MetricValue) {
	if len(metrics) == 0 {
		return
	}

	debug := Config().Debug
	debug_endpoints := Config().Debugmetric.Endpoints
	debug_metrics := Config().Debugmetric.Metrics
	debug_tags := Config().Debugmetric.Tags
	debug_Tags := strings.Split(debug_tags, ",")
	if debug {
		for _, metric := range metrics {
			metric_tags := strings.Split(metric.Tags, ",")
			if in_array(metric.Endpoint, debug_endpoints) && in_array(metric.Metric, debug_metrics) {
				if array_include(debug_Tags, metric_tags) {
					log.Printf("=> <Total=%d> %v\n", len(metrics), metric)
				}
				if debug_tags == "" {
					log.Printf("=> <Total=%d> %v\n", len(metrics), metric)
				}

			}
		}
	}

	var resp model.TransferResponse
	err := TransferClient.Call("Transfer.Update", metrics, &resp)
	if err != nil {
		log.Println("call Transfer.Update fail", err)
	}

	if debug {
		log.Println("<=", &resp)
	}
}

func array_include(array_a []string, array_b []string) bool { //b include a
	for _, v := range array_a {
		if in_array(v, array_b) {
			continue
		} else {
			return false
		}
	}
	return true
}

func in_array(a string, array []string) bool {
	for _, v := range array {
		if a == v {
			return true
		}
	}
	return false
}
