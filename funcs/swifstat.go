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
	AliveIp     []string
	lastIfStat  = map[string]*[]sw.IfStats{}
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
	ignoreSpeedPercent  bool
	fastPingMode        bool
)

func initVariable() {
	pingTimeout = g.Config().Switch.PingTimeout
	fastPingMode = g.Config().Switch.FastPingMode
	pingRetry = g.Config().Switch.PingRetry

	community = g.Config().Switch.Community
	snmpTimeout = g.Config().Switch.SnmpTimeout
	snmpRetry = g.Config().Switch.SnmpRetry

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
	ts := time.Now().Unix()
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
				if ignoreSpeedPercent == false {
					L = append(L, GaugeValueIp(ifStat.TS, ip, "switch.if.Speed", ifStat.IfSpeed, ifNameTag, ifIndexTag))
				}
				if ignoreBroadcastPkt == false {
					if lastIfStatList, ok := lastIfStat[chIfStat.Ip]; ok {
						for _, lastifStat := range *lastIfStatList {
							if ifStat.IfIndex == lastifStat.IfIndex {
								interval := ifStat.TS - lastifStat.TS
								broadcastlimit := g.Config().Switch.BroadcastPktlimit
								IfHCInBroadcastPkts := float64(ifStat.IfHCInBroadcastPkts-lastifStat.IfHCInBroadcastPkts) / float64(interval)
								IfHCOutBroadcastPkts := float64(ifStat.IfHCOutBroadcastPkts-lastifStat.IfHCOutBroadcastPkts) / float64(interval)
								if IfHCInBroadcastPkts >= 0 && IfHCInBroadcastPkts <= broadcastlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.InBroadcastPkt", IfHCInBroadcastPkts, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.InBroadcastPkt ", "out of range, value is ", IfHCInBroadcastPkts, "Limit is ", broadcastlimit)
									log.Println("IfHCInBroadcastPkts This Time: ", ifStat.IfHCInBroadcastPkts)
									log.Println("IfHCInBroadcastPkts Last Time: ", lastifStat.IfHCInBroadcastPkts)
								}
								if IfHCOutBroadcastPkts >= 0 && IfHCOutBroadcastPkts <= broadcastlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.OutBroadcastPkt", IfHCOutBroadcastPkts, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.OutBroadcastPkt ", "out of range, value is ", IfHCOutBroadcastPkts, "Limit is ", broadcastlimit)
									log.Println("IfHCOutBroadcastPkts This Time: ", ifStat.IfHCOutBroadcastPkts)
									log.Println("IfHCOutBroadcastPkts Last Time: ", lastifStat.IfHCOutBroadcastPkts)
								}
							}
						}
					}
				}
				if ignoreMulticastPkt == false {
					if lastIfStatList, ok := lastIfStat[chIfStat.Ip]; ok {
						for _, lastifStat := range *lastIfStatList {
							if ifStat.IfIndex == lastifStat.IfIndex {
								interval := ifStat.TS - lastifStat.TS
								multicastlimit := g.Config().Switch.MulticastPktlimit
								IfHCInMulticastPkts := float64(ifStat.IfHCInMulticastPkts-lastifStat.IfHCInMulticastPkts) / float64(interval)
								IfHCOutMulticastPkts := float64(ifStat.IfHCOutMulticastPkts-lastifStat.IfHCOutMulticastPkts) / float64(interval)
								if IfHCInMulticastPkts >= 0 && IfHCInMulticastPkts <= multicastlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.InMulticastPkt", IfHCInMulticastPkts, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.InMulticastPkt ", "out of range, value is ", IfHCInMulticastPkts, "Limit is ", multicastlimit)
									log.Println("IfHCInMulticastPkts This Time: ", ifStat.IfHCInMulticastPkts)
									log.Println("IfHCInMulticastPkts Last Time: ", lastifStat.IfHCInMulticastPkts)
								}
								if IfHCOutMulticastPkts >= 0 && IfHCOutMulticastPkts <= multicastlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.OutMulticastPkt", IfHCOutMulticastPkts, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.OutMulticastPkt ", "out of range, value is ", IfHCOutMulticastPkts, "Limit is ", multicastlimit)
									log.Println("IfHCOutMulticastPkts This Time: ", ifStat.IfHCOutMulticastPkts)
									log.Println("IfHCOutMulticastPkts Last Time: ", lastifStat.IfHCOutMulticastPkts)
								}
							}
						}
					}
				}

				if ignoreDiscards == false {
					if lastIfStatList, ok := lastIfStat[chIfStat.Ip]; ok {
						for _, lastifStat := range *lastIfStatList {
							if ifStat.IfIndex == lastifStat.IfIndex {
								interval := ifStat.TS - lastifStat.TS
								discardlimit := g.Config().Switch.DiscardsPktlimit
								IfInDiscards := float64(ifStat.IfInDiscards-lastifStat.IfInDiscards) / float64(interval)
								IfOutDiscards := float64(ifStat.IfOutDiscards-lastifStat.IfOutDiscards) / float64(interval)
								if IfInDiscards >= 0 && IfInDiscards <= discardlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.InDiscards", IfInDiscards, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.InDiscards ", "out of range, value is ", IfInDiscards, "Limit is ", discardlimit)
									log.Println("IfInDiscards This Time: ", ifStat.IfInDiscards)
									log.Println("IfInDiscards Last Time: ", lastifStat.IfInDiscards)
								}
								if IfOutDiscards >= 0 && IfOutDiscards <= discardlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.OutDiscards", IfOutDiscards, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.OutDiscards ", "out of range, value is ", IfOutDiscards, "Limit is ", discardlimit)
									log.Println("IfOutDiscards This Time: ", ifStat.IfOutDiscards)
									log.Println("IfOutDiscards Last Time: ", lastifStat.IfOutDiscards)
								}
							}
						}
					}
				}

				if ignoreErrors == false {
					if lastIfStatList, ok := lastIfStat[chIfStat.Ip]; ok {
						for _, lastifStat := range *lastIfStatList {
							if ifStat.IfIndex == lastifStat.IfIndex {
								interval := ifStat.TS - lastifStat.TS
								errorlimit := g.Config().Switch.ErrorsPktlimit
								IfInErrors := float64(ifStat.IfInErrors-lastifStat.IfInErrors) / float64(interval)
								IfOutErrors := float64(ifStat.IfOutErrors-lastifStat.IfOutErrors) / float64(interval)
								if IfInErrors >= 0 && IfInErrors <= errorlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.InErrors", IfInErrors, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.InErrors ", "out of range, value is ", IfInErrors, "Limit is ", errorlimit)
									log.Println("IfInErrors This Time: ", ifStat.IfInErrors)
									log.Println("IfInErrors Last Time: ", lastifStat.IfInErrors)
								}
								if IfOutErrors >= 0 && IfOutErrors <= errorlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.OutErrors", IfOutErrors, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.OutErrors ", "out of range, value is ", IfOutErrors, "Limit is ", errorlimit)
									log.Println("IfOutErrors This Time: ", ifStat.IfOutErrors)
									log.Println("IfOutErrors Last Time: ", lastifStat.IfOutErrors)
								}
							}
						}
					}
				}

				if ignoreUnknownProtos == false {
					if lastIfStatList, ok := lastIfStat[chIfStat.Ip]; ok {
						for _, lastifStat := range *lastIfStatList {
							if ifStat.IfIndex == lastifStat.IfIndex {
								interval := ifStat.TS - lastifStat.TS
								unknownProtoslimit := g.Config().Switch.UnknownProtosPktlimit
								IfInUnknownProtos := float64(ifStat.IfInUnknownProtos-lastifStat.IfInUnknownProtos) / float64(interval)
								if IfInUnknownProtos >= 0 && IfInUnknownProtos <= unknownProtoslimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.InUnknownProtos", IfInUnknownProtos, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.InUnknownProtos ", "out of range, value is ", IfInUnknownProtos, "Limit is ", unknownProtoslimit)
									log.Println("IfOutQLen This Time: ", ifStat.IfInUnknownProtos)
									log.Println("IfOutQLen Last Time: ", lastifStat.IfInUnknownProtos)
								}
							}
						}
					}
				}

				if ignoreOutQLen == false {
					if lastIfStatList, ok := lastIfStat[chIfStat.Ip]; ok {
						for _, lastifStat := range *lastIfStatList {
							if ifStat.IfIndex == lastifStat.IfIndex {
								interval := ifStat.TS - lastifStat.TS
								outQlenlimit := g.Config().Switch.OutQLenPktlimit
								IfOutQLen := float64(ifStat.IfOutQLen-lastifStat.IfOutQLen) / float64(interval)
								if IfOutQLen >= 0 && IfOutQLen <= outQlenlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.OutQLen", IfOutQLen, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.OutQLen ", "out of range, value is ", IfOutQLen, "Limit is ", outQlenlimit)
									log.Println("IfOutQLen This Time: ", ifStat.IfOutQLen)
									log.Println("IfOutQLen Last Time: ", lastifStat.IfOutQLen)
								}
							}
						}
					}
				}

				//如果IgnorePkt为false，采集Pkt
				if ignorePkt == false {
					if lastIfStatList, ok := lastIfStat[chIfStat.Ip]; ok {
						for _, lastifStat := range *lastIfStatList {
							if ifStat.IfIndex == lastifStat.IfIndex {
								interval := ifStat.TS - lastifStat.TS
								pktlimit := g.Config().Switch.Pktlimit
								IfHCInUcastPkts := float64(ifStat.IfHCInUcastPkts-lastifStat.IfHCInUcastPkts) / float64(interval)
								IfHCOutUcastPkts := float64(ifStat.IfHCOutUcastPkts-lastifStat.IfHCOutUcastPkts) / float64(interval)
								if IfHCInUcastPkts >= 0 && IfHCInUcastPkts <= pktlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.InPkts", IfHCInUcastPkts, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.InPkts ", "out of range, value is ", IfHCInUcastPkts, "Limit is ", pktlimit)
									log.Println("IfHCInUcastPkts This Time: ", ifStat.IfHCInUcastPkts)
									log.Println("IfHCInUcastPkts Last Time: ", lastifStat.IfHCInUcastPkts)
								}
								if IfHCOutUcastPkts >= 0 && IfHCOutUcastPkts <= pktlimit {
									L = append(L, GaugeValueIp(ts, ip, "switch.if.OutPkts", IfHCOutUcastPkts, ifNameTag, ifIndexTag))
								} else {
									log.Println(ip, ifNameTag, "switch.if.OutPkts ", "out of range, value is ", IfHCOutUcastPkts, "Limit is ", pktlimit)
									log.Println("IfHCOutUcastPkts This Time: ", ifStat.IfHCOutUcastPkts)
									log.Println("IfHCOutUcastPkts Last Time: ", lastifStat.IfHCOutUcastPkts)
								}
							}
						}
					}
				}
				if lastIfStatList, ok := lastIfStat[chIfStat.Ip]; ok {
					for _, lastifStat := range *lastIfStatList {
						if ifStat.IfIndex == lastifStat.IfIndex {
							interval := ifStat.TS - lastifStat.TS
							speedlimit := g.Config().Switch.Speedlimit
							if speedlimit == 0 {
								speedlimit = float64(ifStat.IfSpeed)
							}
							IfHCInOctets := 8 * float64(ifStat.IfHCInOctets-lastifStat.IfHCInOctets) / float64(interval)
							IfHCOutOctets := 8 * float64(ifStat.IfHCOutOctets-lastifStat.IfHCOutOctets) / float64(interval)
							if IfHCInOctets >= 0 && IfHCInOctets <= speedlimit {
								InSpeedPercent := IfHCInOctets / float64(ifStat.IfSpeed)
								L = append(L, GaugeValueIp(ts, ip, "switch.if.In", IfHCInOctets, ifNameTag, ifIndexTag))
								L = append(L, GaugeValueIp(ts, ip, "switch.if.InSpeedPercent", InSpeedPercent, ifNameTag, ifIndexTag))
							} else {
								log.Println(ip, ifNameTag, "switch.if.In ", "out of range, value is ", IfHCInOctets, "Limit is ", speedlimit)
								log.Println("IfHCInOctets This Time: ", ifStat.IfHCInOctets)
								log.Println("IfHCInOctets Last Time: ", lastifStat.IfHCInOctets)
							}
							if IfHCOutOctets >= 0 && IfHCOutOctets <= speedlimit {
								OutSpeedPercent := IfHCOutOctets / float64(ifStat.IfSpeed)
								L = append(L, GaugeValueIp(ts, ip, "switch.if.Out", IfHCOutOctets, ifNameTag, ifIndexTag))
								L = append(L, GaugeValueIp(ts, ip, "switch.if.OutSpeedPercent", OutSpeedPercent, ifNameTag, ifIndexTag))
							} else {
								log.Println(ip, ifNameTag, "switch.if.Out ", "out of range, value is ", IfHCOutOctets, "Limit is ", speedlimit)
								log.Println("IfHCOutOctets This Time: ", ifStat.IfHCOutOctets)
								log.Println("IfHCOutOctets Last Time: ", lastifStat.IfHCOutOctets)
							}
						}
					}
				}
			}
		}
		lastIfStat[chIfStat.Ip] = chIfStat.IfStatsList
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
