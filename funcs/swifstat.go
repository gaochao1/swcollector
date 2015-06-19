package funcs

import (
	"github.com/gaochao1/swcollector/g"
	"github.com/open-falcon/common/model"
	"log"

	"github.com/gaochao1/sw"
	"github.com/toolkits/slice"

	"strconv"
	"time"
)

type ChIfStat struct {
	Ip          string
	PingResult  bool
	UseTime     int64
	IfStatsList *[]sw.IfStats
}

var (
	IpList  = make([]string, 3000, 5000)
	AliveIp []string
)

func AllSwitchIp() []string {
	var allIp []string
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

func SwIfMetrics() (L []*model.MetricValue) {
	if g.Config().Switch.Enabled && len(g.Config().Switch.IpRange) > 0 {
		return swIfMetrics()
	}
	return
}

func swIfMetrics() (L []*model.MetricValue) {
	allIp := AllSwitchIp()

	chs := make([]chan ChIfStat, len(allIp))
	limitCh := make(chan bool, g.Config().Switch.LimitConcur)

	startTime := time.Now()
	log.Println("INFO : UpdateIfStats the maximum number of concurrent limited to :", strconv.Itoa(g.Config().Switch.LimitConcur), "Number of all switch ip : ", len(allIp))

	for i, ip := range allIp {
		chs[i] = make(chan ChIfStat)
		limitCh <- true
		go coreSwIfMetrics(ip, chs[i], limitCh)
		time.Sleep(5 * time.Millisecond)
	}

	for _, ch := range chs {
		chIfStat := <-ch

		if chIfStat.PingResult == true && !slice.ContainsString(AliveIp, chIfStat.Ip) {
			AliveIp = append(AliveIp, chIfStat.Ip)
		}

		if chIfStat.IfStatsList != nil {

			if g.Config().Debug {
				log.Println(chIfStat.Ip, chIfStat.PingResult, len(*chIfStat.IfStatsList), chIfStat.UseTime)
			}

			for _, ifStat := range *chIfStat.IfStatsList {
				ifNameTag := "ifName=" + ifStat.IfName
				ifIndexTag := "ifIndex=" + strconv.Itoa(ifStat.IfIndex)
				ip := chIfStat.Ip

				L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.In", ifStat.IfHCInOctets, ifNameTag, ifIndexTag))
				L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.Out", ifStat.IfHCOutOctets, ifNameTag, ifIndexTag))

				//如果IgnorePkt为false，采集Pkt
				if g.Config().Switch.IgnorePkt == false {
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.InPkts", ifStat.IfHCInUcastPkts, ifNameTag, ifIndexTag))
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.OutPkts", ifStat.IfHCOutUcastPkts, ifNameTag, ifIndexTag))
				}

			}
		}
	}

	endTime := time.Now()
	log.Println("INFO : UpdateIfStats complete. Process time :", endTime.Sub(startTime), "Number of live switch :", len(AliveIp))

	if g.Config().Debug {
		for i, v := range AliveIp {
			log.Println("AliveIp:", i, v)
		}
	}

	return L
}

func coreSwIfMetrics(ip string, ch chan ChIfStat, limitCh chan bool) {
	var startTime, endTime int64
	startTime = time.Now().Unix()

	var chIfStat ChIfStat
	var pingResult bool

	for i := 0; i < g.Config().Switch.PingRetry; i++ {
		pingResult = sw.Ping(ip, g.Config().Switch.PingTimeout)
		if pingResult == true {
			break
		}
	}

	chIfStat.Ip = ip
	chIfStat.PingResult = pingResult

	if !pingResult {
		endTime = time.Now().Unix()
		chIfStat.UseTime = (endTime - startTime)
		<-limitCh
		ch <- chIfStat
		return
	} else {
		var ifList []sw.IfStats
		var err error

		vendor, _ := sw.SysVendor(ip, g.Config().Switch.Community, g.Config().Switch.SnmpTimeout)
		if vendor == "Huawei" {
			ifList, err = sw.ListIfStatsSnmpWalk(ip, g.Config().Switch.Community, g.Config().Switch.SnmpTimeout*5, g.Config().Switch.IgnoreIface, g.Config().Switch.SnmpRetry, g.Config().Switch.IgnorePkt)
		} else {
			ifList, err = sw.ListIfStats(ip, g.Config().Switch.Community, g.Config().Switch.SnmpTimeout, g.Config().Switch.IgnoreIface, g.Config().Switch.SnmpRetry, g.Config().Switch.IgnorePkt)
		}
		if err != nil || len(ifList) == 0 {
			endTime = time.Now().Unix()
			chIfStat.UseTime = (endTime - startTime)
			<-limitCh
			ch <- chIfStat
			return
		}

		for i := 0; i < len(ifList); i++ {
			chIfStat.IfStatsList = &ifList
		}

		endTime = time.Now().Unix()
		chIfStat.UseTime = (endTime - startTime)
		<-limitCh
		ch <- chIfStat
		return
	}

	endTime = time.Now().Unix()
	chIfStat.UseTime = (endTime - startTime)
	<-limitCh
	ch <- chIfStat
	return
}
