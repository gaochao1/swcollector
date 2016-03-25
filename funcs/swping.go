package funcs

import (
	"log"
	"time"

	"github.com/baishancloud/swcollector/g"
	"github.com/gaochao1/sw"
	"github.com/open-falcon/common/model"
)

type SwPing struct {
	Ip   string
	Ping float64
}

func PingMetrics() (L []*model.MetricValue) {

	chs := make([]chan SwPing, len(AliveIp))
	for i, ip := range AliveIp {
		if ip != "" {
			chs[i] = make(chan SwPing)
			go pingMetrics(ip, chs[i])
		}
	}

	for _, ch := range chs {
		swPing := <-ch
		L = append(L, GaugeValueIp(time.Now().Unix(), swPing.Ip, "switch.Ping", swPing.Ping))
	}

	return L
}

func pingMetrics(ip string, ch chan SwPing) {
	var swPing SwPing
	timeout := g.Config().Switch.PingTimeout * 4

	rtt, err := sw.PingRtt(ip, timeout)
	if err != nil {
		log.Println(ip, err)
		swPing.Ping = 0
	}

	swPing.Ip = ip
	swPing.Ping = rtt
	ch <- swPing

	return
}
