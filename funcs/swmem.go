package funcs

import (
	"log"
	"time"

	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/gaochao1/sw"
	"github.com/open-falcon/common/model"
)

type SwMem struct {
	Ip       string
	MemUtili int
}

func MemMetrics() (L []*model.MetricValue) {

	chs := make([]chan SwMem, len(AliveIp))
	for i, ip := range AliveIp {
		if ip != "" {
			chs[i] = make(chan SwMem)
			go memMetrics(ip, chs[i])
		}
	}

	for _, ch := range chs {
		swMem := <-ch
		ipTag := "ip=" + swMem.Ip
		L = append(L, GaugeValueIp(time.Now().Unix(), swMem.Ip, "switch.MemUtilization", swMem.MemUtili, ipTag))
	}

	return L
}

func memMetrics(ip string, ch chan SwMem) {
	var swMem SwMem

	memUtili, err := sw.MemUtilization(ip, g.Config().Switch.Community, g.Config().Switch.SnmpTimeout, g.Config().Switch.SnmpRetry)
	if err != nil {
		log.Println(err)
	}

	swMem.Ip = ip
	swMem.MemUtili = memUtili
	ch <- swMem

	return
}
