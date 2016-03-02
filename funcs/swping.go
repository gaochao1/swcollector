package funcs

import (
	"github.com/gaochao1/sw"
	"github.com/gaochao1/swcollector/g"
	"github.com/open-falcon/common/model"
	"log"
	"time"
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
	timeout := g.Config().Switch.PingTimeout * g.Config().Switch.PingRetry

	rtt, err := sw.PingRtt(ip, timeout)
	if err != nil {
		log.Println(ip, err)
		swPing.Ip = ip
		swPing.Ping = -1
		ch <- swPing
		return
	}
	if g.Config().Debug {
		log.Println(ip, rtt)
	}
	swPing.Ip = ip
	swPing.Ping = rtt
	ch <- swPing
	return

}
