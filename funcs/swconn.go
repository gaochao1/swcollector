package funcs

import (
	"github.com/gaochao1/sw"
	"github.com/gaochao1/swcollector/g"
	"github.com/open-falcon/common/model"
	"log"
	"strings"
	"time"
)

type SwConn struct {
	Ip             string
	ConnectionStat int
}

func ConnMetrics() (L []*model.MetricValue) {

	chs := make([]chan SwConn, len(AliveIp))
	for i, ip := range AliveIp {
		if ip != "" {
			chs[i] = make(chan SwConn)
			go connMetrics(ip, chs[i])
		}
	}

	for _, ch := range chs {
		swConn := <-ch
		L = append(L, GaugeValueIp(time.Now().Unix(), swConn.Ip, "switch.ConnectionStat", swConn.ConnectionStat))
	}

	return L
}

func connMetrics(ip string, ch chan SwConn) {
	var swConn SwConn
	vendor, _ := sw.SysVendor(ip, community, snmpTimeout)
	if !strings.Contains(vendor, "Cisco_ASA") {
		ch <- swConn
		return
	}
	ConnectionStat, err := sw.ConnectionStat(ip, g.Config().Switch.Community, g.Config().Switch.SnmpTimeout, g.Config().Switch.SnmpRetry)
	if err != nil {
		log.Println(err)
	}

	swConn.Ip = ip
	swConn.ConnectionStat = ConnectionStat
	ch <- swConn

	return
}
