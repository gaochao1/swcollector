package lansw

import (
	"fmt"
	"log"
	"time"

	"github.com/baishancloud/octopux-swcollector/funcs"
	"github.com/baishancloud/octopux-swcollector/g"

	pfc "github.com/baishancloud/goperfcounter"
	"github.com/gaochao1/gosnmp"
	"github.com/gaochao1/sw"
	"github.com/getsentry/raven-go"
	"github.com/open-falcon/common/model"
)

const (
	ifNameOid       = "1.3.6.1.2.1.31.1.1.1.1"
	ifNameOidPrefix = ".1.3.6.1.2.1.31.1.1.1.1."

	ifHCInOid       = "1.3.6.1.2.1.31.1.1.1.6."
	ifHCInOidPrefix = ".1.3.6.1.2.1.31.1.1.1.6."
	ifHCOutOid      = "1.3.6.1.2.1.31.1.1.1.10."

	ifHCInPktsOid       = "1.3.6.1.2.1.31.1.1.1.7."
	ifHCInPktsOidPrefix = ".1.3.6.1.2.1.31.1.1.1.7."
	ifHCOutPktsOid      = "1.3.6.1.2.1.31.1.1.1.11."

	ifInDiscardpktsOid       = "1.3.6.1.2.1.2.2.1.13."
	ifInDiscardpktsOidPrefix = ".1.3.6.1.2.1.2.2.1.13."
	ifOutDiscardpktsOid      = "1.3.6.1.2.1.2.2.1.19."
)

func getUint64(value interface{}) uint64 {
	switch v := value.(type) {
	case int:
		return uint64(v)
	case int32:
		return uint64(v)
	case int64:
		return uint64(v)
	case uint:
		return uint64(v)
	case uint64:
		return uint64(v)
	case uint32:
		return uint64(v)
	default:
		log.Printf("not int type :%V,%v", v, v)
		return uint64(0)
	}
}

func handleSnmpTask(task *SnmpTask) bool {

	defer func() {
		if r := recover(); r != nil {
			raven.CaptureError(fmt.Errorf("Recovered in ListIfStats %s:%s , %v", task.Ip, task.Ifname, r), nil)
			log.Println(task.Ip, task.Ifname, " Recovered in ListIfStats", r)
		}
	}()

	chIfInList := make(chan []gosnmp.SnmpPDU)
	chIfOutList := make(chan []gosnmp.SnmpPDU)

	chIfInPktList := make(chan []gosnmp.SnmpPDU)
	chIfOutPktList := make(chan []gosnmp.SnmpPDU)

	chIfInDcPktList := make(chan []gosnmp.SnmpPDU)
	chIfOutDcPktList := make(chan []gosnmp.SnmpPDU)
	pfc.Meter("SWCLSnmpCnt", int64(1))
	go IfHCInOctets(task, community, snmpTimeout, chIfInList, snmpRetry)
	go IfHCOutOctets(task, community, snmpTimeout, chIfOutList, snmpRetry)
	ifOutList := <-chIfOutList
	now := time.Now().Unix()
	ifInList := <-chIfInList

	var ifInPktList, ifOutPktList []gosnmp.SnmpPDU
	var ifInDcPktList, ifOutDcPktList []gosnmp.SnmpPDU

	pktcol := checkIgnorePkt()

	if pktcol {

		go IfHCInUcastPkts(task, community, snmpTimeout, chIfInPktList, snmpRetry)
		go IfHCOutUcastPkts(task, community, snmpTimeout, chIfOutPktList, snmpRetry)
		ifInPktList = <-chIfInPktList
		ifOutPktList = <-chIfOutPktList
		go IfInDiscardPkts(task, community, snmpTimeout, chIfInDcPktList, snmpRetry)
		go IfOutDiscardPkts(task, community, snmpTimeout, chIfOutDcPktList, snmpRetry)
		ifInDcPktList = <-chIfInDcPktList
		ifOutDcPktList = <-chIfOutDcPktList
	}
	if len(ifInList) > 0 || len(ifOutList) > 0 {
		ifStat := &IfStats{}
		ifStat.IfName = task.Ifname
		ifStat.IfIndex = task.Ifindex
		if len(ifInList) > 0 {
			//log.Println(task.Ip, task.Ifname, "in", ifInList[0].Type.String())

			ifStat.IfHCInOctets = getUint64(ifInList[0].Value)

		}
		if len(ifOutList) > 0 {
			//log.Println(task.Ip, task.Ifname, "out", ifOutList[0].Type.String())
			ifStat.IfHCOutOctets = getUint64(ifOutList[0].Value)
		}
		if pktcol {
			if len(ifInPktList) > 0 {
				ifStat.IfHCInUcastPkts = getUint64(ifInPktList[0].Value)
			}
			if len(ifOutPktList) > 0 {
				ifStat.IfHCOutUcastPkts = getUint64(ifOutPktList[0].Value)
			}
			if len(ifInDcPktList) > 0 {
				ifStat.IfInDiscardPkts = getUint64(ifInDcPktList[0].Value)
			}
			if len(ifOutDcPktList) > 0 {
				ifStat.IfOutDiscardPkts = getUint64(ifOutDcPktList[0].Value)
			}
		}
		ifStat.TS = now
		issucess := Push2SendQueue(task, ifStat)
		if !issucess && isdebug {
			pfc.Meter("SWCLSnmpError", int64(1))
			log.Println(task.Ip, task.Ifname, " push IfStatsQueue false!")
		}
		return issucess

	} else {
		pfc.Meter("SWCLSnmpError", int64(1))
		log.Println(task.Ip, task.Ifname, " snmp get IfStatsQueue false len = 0!")
		return false
	}
}

func Push2SendQueue(task *SnmpTask, ifstat *IfStats) bool {
	ifNameTag := "ifName=" + task.Ifname
	ifIndexTag := "ifIndex=" + task.Ifindex
	ip := task.Ip
	ipTag := "ip=" + ip
	var L []*model.MetricValue
	if ifstat.IfHCInOctets > 0 {
		L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.In", ifstat.IfHCInOctets, ifNameTag, ifIndexTag, ipTag))
		rate, err := g.Rate(task.Ip, task.Ifname, "swin", ifstat.IfHCInOctets, ifstat.TS)
		if err == nil && enableRate {
			L = append(L, funcs.GaugeValueIp(ifstat.TS, ip, "swinrate", rate, ifNameTag, ifIndexTag, ipTag))
		}
	}
	if ifstat.IfHCOutOctets > 0 {
		L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.Out", ifstat.IfHCOutOctets, ifNameTag, ifIndexTag, ipTag))
		rate, err := g.Rate(task.Ip, task.Ifname, "swout", ifstat.IfHCOutOctets, ifstat.TS)
		if err == nil && enableRate {
			L = append(L, funcs.GaugeValueIp(ifstat.TS, ip, "swoutrate", rate, ifNameTag, ifIndexTag, ipTag))
		}
	}
	if !ignorePkt {
		if ifstat.IfHCInUcastPkts > 0 {
			L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.InPkts", ifstat.IfHCInUcastPkts, ifNameTag, ifIndexTag, ipTag))
			rate, err := g.Rate(task.Ip, task.Ifname, "inpkts", ifstat.IfHCInUcastPkts, ifstat.TS)
			if err == nil && enableRate {
				L = append(L, funcs.GaugeValueIp(ifstat.TS, ip, "swinpktsrate", rate, ifNameTag, ifIndexTag, ipTag))
			}
		}
		if ifstat.IfHCOutUcastPkts > 0 {
			L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.OutPkts", ifstat.IfHCOutUcastPkts, ifNameTag, ifIndexTag, ipTag))
			rate, err := g.Rate(task.Ip, task.Ifname, "outpkts", ifstat.IfHCOutUcastPkts, ifstat.TS)
			if err == nil && enableRate {
				L = append(L, funcs.GaugeValueIp(ifstat.TS, ip, "swoutpktsrate", rate, ifNameTag, ifIndexTag, ipTag))
			}
		}
		if ifstat.IfInDiscardPkts > 0 {
			L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.InDisCardPkts", ifstat.IfInDiscardPkts, ifNameTag, ifIndexTag, ipTag))
			rate, err := g.Rate(task.Ip, task.Ifname, "indcpkts", ifstat.IfInDiscardPkts, ifstat.TS)
			if err == nil && enableRate {
				L = append(L, funcs.GaugeValueIp(ifstat.TS, ip, "swindcpktsrate", rate, ifNameTag, ifIndexTag, ipTag))
			}
		}
		if ifstat.IfOutDiscardPkts > 0 {
			L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.OutDiscardPkts", ifstat.IfOutDiscardPkts, ifNameTag, ifIndexTag, ipTag))
			rate, err := g.Rate(task.Ip, task.Ifname, "outdcpkts", ifstat.IfOutDiscardPkts, ifstat.TS)
			if err == nil && enableRate {
				L = append(L, funcs.GaugeValueIp(ifstat.TS, ip, "swoutdcpktsrate", rate, ifNameTag, ifIndexTag, ipTag))
			}
		}
	}
	if len(L) == 0 {
		return false
	}
	var interfaceSlice []interface{} = make([]interface{}, len(L))
	for i, d := range L {
		interfaceSlice[i] = d
	}
	return IfstatsQueue.PushFrontBatch(interfaceSlice)

}
func ListIfName(ip, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(ip, community, timeout, ch, retry, ifNameOid, "walk")
}

func IfInDiscardPkts(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifInDiscardpktsOid+task.Ifindex, "get")
}

func IfOutDiscardPkts(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifOutDiscardpktsOid+task.Ifindex, "get")
}

func IfHCInOctets(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifHCInOid+task.Ifindex, "get")
}

func IfHCOutOctets(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifHCOutOid+task.Ifindex, "get")
}

func IfHCInUcastPkts(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifHCInPktsOid+task.Ifindex, "get")
}

func IfHCOutUcastPkts(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifHCOutPktsOid+task.Ifindex, "get")
}

func RunSnmpRetry(ip, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int, oid string, method string) {
	//method := "get"
	var snmpPDUs []gosnmp.SnmpPDU
	var err error
	for i := 0; i < retry; i++ {
		snmpPDUs, err = sw.RunSnmp(ip, community, oid, method, timeout)
		if len(snmpPDUs) > 0 {
			ch <- snmpPDUs
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		//raven.CaptureError(err, nil)
		//pfc.Meter("SWCLSnmpTW", int64(1))
		log.Println("RunSnmpRetry error:", ip, oid, method, timeout, err.Error())
	}
	ch <- []gosnmp.SnmpPDU{}
	return
}

func checkIgnorePkt() bool {
	if ignorePkt {
		return !ignorePkt
	}
	unixtime := time.Now().Unix()
	intervalsec := int64(interval.Seconds())
	return (unixtime/intervalsec)%5 == 0
}
