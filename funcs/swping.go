package funcs

import (
	"github.com/gaochao1/sw"
	"github.com/gaochao1/swcollector/g"
	"github.com/open-falcon/common/model"
	"log"
	"strconv"
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
		log.Println("===", swPing)
		L = append(L, GaugeValueIp(time.Now().Unix(), swPing.Ip, "switch.Ping", swPing.Ping))
	}

	return L
}

func pingMetrics(ip string, ch chan SwPing) {
	var swPing SwPing

	ping, err := sw.PingStatSummary(ip, g.Config().Switch.PingRetry, g.Config().Switch.PingTimeout)
	if err != nil {
		log.Println(err)
	}

	swPing.Ip = ip
	swPing.Ping, _ = strconv.ParseFloat(ping["avg"], 64)
	ch <- swPing

	return
}
