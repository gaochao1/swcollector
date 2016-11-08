package funcs

import (
	"log"
	"time"

	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/gaochao1/sw"
	"github.com/open-falcon/common/model"
	"github.com/toolkits/slice"
)

var (
	AliveIp []string
)

type SwPing struct {
	Ip   string
	Ping float64
}

func AllSwitchIp() (allIp []string) {
	switchIp := g.Config().Switch.IpRange

	if len(switchIp) > 0 {
		for _, sip := range switchIp {
			aip := sw.ParseIp(sip)
			for _, ip := range aip {
				allIp = append(allIp, ip)
			}
		}
	}
	return allIp
}

func PingMetrics() (L []*model.MetricValue) {
	allip := AllSwitchIp()
	chs := make([]chan SwPing, len(allip))
	for i, ip := range allip {
		if ip != "" {
			chs[i] = make(chan SwPing)
			go pingMetrics(ip, chs[i])
		}
	}

	for _, ch := range chs {
		swPing := <-ch

		if swPing.Ping > 0 && !slice.ContainsString(AliveIp, swPing.Ip) {
			AliveIp = append(AliveIp, swPing.Ip)
		}
		ipTag := "ip=" + swPing.Ip
		L = append(L, GaugeValueIp(time.Now().Unix(), swPing.Ip, "switch.Ping", swPing.Ping, ipTag))
	}

	return L
}

func pingMetrics(ip string, ch chan SwPing) {
	var swPing SwPing
	timeout := g.Config().Switch.PingTimeout * 4

	rtt, err := sw.PingRtt(ip, timeout, true)
	if err != nil {
		log.Println(ip, err)
		swPing.Ping = 0
	}

	swPing.Ip = ip
	swPing.Ping = rtt
	ch <- swPing

	return
}
