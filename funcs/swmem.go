package funcs

import (
	"github.com/gaochao1/sw"
	"github.com/gaochao1/swcollector/g"
	"github.com/open-falcon/common/model"
	"log"
	"time"
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
		L = append(L, GaugeValueIp(time.Now().Unix(), swMem.Ip, "switch.MemUtilization", swMem.MemUtili))
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
