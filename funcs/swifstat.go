package funcs

import (
	"log"

	"github.com/gaochao1/swcollector/g"
	"github.com/open-falcon/common/model"

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
	AliveIp []string

	pingTimeout int
	pingRetry   int

	community           string
	snmpTimeout         int
	snmpRetry           int
	displayByBit        bool
	gosnmp              bool
	ignoreIface         []string
	ignorePkt           bool
	ignoreBroadcastPkt  bool
	ignoreMulticastPkt  bool
	ignoreDiscards      bool
	ignoreErrors        bool
	ignoreOperStatus    bool
	ignoreUnknownProtos bool
	ignoreOutQLen       bool
	fastPingMode        bool
)

func initVariable() {
	pingTimeout = g.Config().Switch.PingTimeout
	fastPingMode = g.Config().Switch.FastPingMode
	pingRetry = g.Config().Switch.PingRetry

	community = g.Config().Switch.Community
	snmpTimeout = g.Config().Switch.SnmpTimeout
	snmpRetry = g.Config().Switch.SnmpRetry

	displayByBit = g.Config().Switch.DisplayByBit
	gosnmp = g.Config().Switch.Gosnmp
	ignoreIface = g.Config().Switch.IgnoreIface
	ignorePkt = g.Config().Switch.IgnorePkt
	ignoreOperStatus = g.Config().Switch.IgnoreOperStatus
	ignoreBroadcastPkt = g.Config().Switch.IgnoreBroadcastPkt
	ignoreMulticastPkt = g.Config().Switch.IgnoreMulticastPkt
	ignoreDiscards = g.Config().Switch.IgnoreDiscards
	ignoreErrors = g.Config().Switch.IgnoreErrors
	ignoreUnknownProtos = g.Config().Switch.IgnoreUnknownProtos
	ignoreOutQLen = g.Config().Switch.IgnoreOutQLen
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

func SwIfMetrics() (L []*model.MetricValue) {
	if g.Config().Switch.Enabled && len(g.Config().Switch.IpRange) > 0 {
		return swIfMetrics()
	}
	return
}

func swIfMetrics() (L []*model.MetricValue) {
	initVariable()

	allIp := AllSwitchIp()

	chs := make([]chan ChIfStat, len(allIp))
	limitCh := make(chan bool, g.Config().Switch.LimitConcur)

	startTime := time.Now()
	log.Printf("UpdateIfStats start. The number of concurrent limited to %d. IP addresses number is %d", g.Config().Switch.LimitConcur, len(allIp))
	if gosnmp {
		log.Println("get snmp message by gosnmp")
	} else {
		log.Println("get snmp message by snmpwalk")
	}
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
				log.Println("IP:", chIfStat.Ip, "PingResult:", chIfStat.PingResult, "len_list:", len(*chIfStat.IfStatsList), "UsedTime:", chIfStat.UseTime)
			}

			for _, ifStat := range *chIfStat.IfStatsList {
				ifNameTag := "ifName=" + ifStat.IfName
				ifIndexTag := "ifIndex=" + strconv.Itoa(ifStat.IfIndex)
				ip := chIfStat.Ip
				if ignoreOperStatus == false {
					L = append(L, GaugeValueIp(ifStat.TS, ip, "switch.if.OperStatus", ifStat.IfOperStatus, ifNameTag, ifIndexTag))
				}
				if ignoreBroadcastPkt == false {
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.InBroadcastPkt", ifStat.IfHCInBroadcastPkts, ifNameTag, ifIndexTag))
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.OutBroadcastPkt", ifStat.IfHCOutBroadcastPkts, ifNameTag, ifIndexTag))
				}
				if ignoreMulticastPkt == false {
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.InMulticastPkt", ifStat.IfHCInMulticastPkts, ifNameTag, ifIndexTag))
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.OutMulticastPkt", ifStat.IfHCOutMulticastPkts, ifNameTag, ifIndexTag))
				}

				if ignoreDiscards == false {
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.InDiscards", ifStat.IfInDiscards, ifNameTag, ifIndexTag))
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.OutDiscards", ifStat.IfOutDiscards, ifNameTag, ifIndexTag))
				}

				if ignoreErrors == false {
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.InErrors", ifStat.IfInErrors, ifNameTag, ifIndexTag))
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.OutErrors", ifStat.IfOutErrors, ifNameTag, ifIndexTag))
				}

				if ignoreUnknownProtos == false {
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.InUnknownProtos", ifStat.IfInUnknownProtos, ifNameTag, ifIndexTag))
				}

				if ignoreOutQLen == false {
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.OutQLen", ifStat.IfOutQLen, ifNameTag, ifIndexTag))
				}

				if displayByBit == true {
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.In", 8*ifStat.IfHCInOctets, ifNameTag, ifIndexTag))
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.Out", 8*ifStat.IfHCOutOctets, ifNameTag, ifIndexTag))
				} else {
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.In", ifStat.IfHCInOctets, ifNameTag, ifIndexTag))
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.Out", ifStat.IfHCOutOctets, ifNameTag, ifIndexTag))

				}
				//如果IgnorePkt为false，采集Pkt
				if ignorePkt == false {
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.InPkts", ifStat.IfHCInUcastPkts, ifNameTag, ifIndexTag))
					L = append(L, CounterValueIp(ifStat.TS, ip, "switch.if.OutPkts", ifStat.IfHCOutUcastPkts, ifNameTag, ifIndexTag))
				}

			}
		}
	}

	endTime := time.Now()
	log.Printf("UpdateIfStats complete. Process time %s. Number of active ip is %d", endTime.Sub(startTime), len(AliveIp))

	if g.Config().Debug {
		for i, v := range AliveIp {
			log.Println("AliveIp:", i, v)
		}
	}

	return
}

func pingCheck(ip string) bool {
	var pingResult bool
	for i := 0; i < pingRetry; i++ {
		pingResult = sw.Ping(ip, pingTimeout, fastPingMode)
		if pingResult == true {
			break
		}
	}
	return pingResult
}

func coreSwIfMetrics(ip string, ch chan ChIfStat, limitCh chan bool) {
	var startTime, endTime int64
	startTime = time.Now().Unix()

	var chIfStat ChIfStat

	pingResult := pingCheck(ip)

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

		if gosnmp {
			ifList, err = sw.ListIfStats(ip, community, snmpTimeout, ignoreIface, snmpRetry, ignorePkt, ignoreOperStatus, ignoreBroadcastPkt, ignoreMulticastPkt, ignoreDiscards, ignoreErrors, ignoreUnknownProtos, ignoreOutQLen)
		} else {
			ifList, err = sw.ListIfStatsSnmpWalk(ip, community, snmpTimeout*5, ignoreIface, snmpRetry, ignorePkt, ignoreOperStatus, ignoreBroadcastPkt, ignoreMulticastPkt, ignoreDiscards, ignoreErrors, ignoreUnknownProtos, ignoreOutQLen)
		}

		if err != nil {
			log.Printf(ip, err)
		}

		if len(ifList) > 0 {
			chIfStat.IfStatsList = &ifList
		}

		endTime = time.Now().Unix()
		chIfStat.UseTime = (endTime - startTime)
		<-limitCh
		ch <- chIfStat
		return
	}

	return
}
