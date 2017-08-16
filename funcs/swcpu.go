package funcs

import (
	"log"
	"time"

	"github.com/gaochao1/sw"
	"github.com/gaochao1/swcollector/g"
	"github.com/open-falcon/common/model"
)

type SwCpu struct {
	Ip      string
	CpuUtil int
}

func CpuMetrics() (L []*model.MetricValue) {

	chs := make([]chan SwCpu, len(AliveIp))
	for i, ip := range AliveIp {
		if ip != "" {
			chs[i] = make(chan SwCpu)
			go cpuMetrics(ip, chs[i])
		}
	}

	for _, ch := range chs {
		swCpu, ok := <-ch
		if !ok {
			continue
		}
		L = append(L, GaugeValueIp(time.Now().Unix(), swCpu.Ip, "switch.CpuUtilization", swCpu.CpuUtil))
	}

	return L
}

func cpuMetrics(ip string, ch chan SwCpu) {
	var swCpu SwCpu

	cpuUtili, err := sw.CpuUtilization(ip, g.Config().Switch.Community, g.Config().Switch.SnmpTimeout, g.Config().Switch.SnmpRetry)
	if err != nil {
		log.Println(err)
		close(ch)
		return
	}

	swCpu.Ip = ip
	swCpu.CpuUtil = cpuUtili
	ch <- swCpu

	return
}
