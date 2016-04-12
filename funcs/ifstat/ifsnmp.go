package ifstat

import (
	"bytes"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/baishancloud/octopux-swcollector/funcs"

	"github.com/gaochao1/gosnmp"
	"github.com/gaochao1/sw"
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

func handleSnmpTask(task *SnmpTask) bool {
	if task.SnmpWalk {
		return coreHandleSnmpTaskWalk(task)
	} else {
		return coreHandleSnmpTask(task)
	}

}

func coreHandleSnmpTask(task *SnmpTask) bool {

	defer func() {
		if r := recover(); r != nil {
			log.Println(task.Ip+" Recovered in ListIfStats", r)
		}
	}()

	chIfInList := make(chan []gosnmp.SnmpPDU)
	chIfOutList := make(chan []gosnmp.SnmpPDU)

	chIfInPktList := make(chan []gosnmp.SnmpPDU)
	chIfOutPktList := make(chan []gosnmp.SnmpPDU)

	chIfInDcPktList := make(chan []gosnmp.SnmpPDU)
	chIfOutDcPktList := make(chan []gosnmp.SnmpPDU)

	go IfHCInOctets(task, community, snmpTimeout, chIfInList, snmpRetry)
	go IfHCOutOctets(task, community, snmpTimeout, chIfOutList, snmpRetry)
	ifInList := <-chIfInList
	ifOutList := <-chIfOutList
	now := time.Now().Unix()

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
	if len(ifInList) > 0 && len(ifOutList) > 0 {
		ifStat := &IfStats{}
		ifStat.IfName = task.Ifname
		ifStat.IfIndex = task.Ifindex
		ifStat.IfHCInOctets = ifInList[0].Value.(uint64)
		ifStat.IfHCOutOctets = ifOutList[0].Value.(uint64)
		if pktcol {
			ifStat.IfHCInUcastPkts = ifInPktList[0].Value.(uint64)
			ifStat.IfHCOutUcastPkts = ifOutPktList[0].Value.(uint64)
			ifStat.IfInDiscardPkts = ifInDcPktList[0].Value.(uint64)
			ifStat.IfOutDiscardPkts = ifOutDcPktList[0].Value.(uint64)
		}
		ifStat.TS = now
		issucess := Push2SendQueue(task, ifStat)
		if !issucess && isdebug {
			log.Println(task.Ip + " push IfStatsQueue false!")
		}
		return issucess

	} else {
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
	}
	if ifstat.IfHCOutOctets > 0 {
		L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.Out", ifstat.IfHCOutOctets, ifNameTag, ifIndexTag, ipTag))
	}
	if !ignorePkt {
		if ifstat.IfHCInUcastPkts > 0 {
			L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.InPkts", ifstat.IfHCInUcastPkts, ifNameTag, ifIndexTag, ipTag))
		}
		if ifstat.IfHCOutUcastPkts > 0 {
			L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.OutPkts", ifstat.IfHCOutUcastPkts, ifNameTag, ifIndexTag, ipTag))
		}
		if ifstat.IfInDiscardPkts > 0 {
			L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.InDisCardPkts", ifstat.IfInDiscardPkts, ifNameTag, ifIndexTag, ipTag))
		}
		if ifstat.IfOutDiscardPkts > 0 {
			L = append(L, funcs.CounterValueIp(ifstat.TS, ip, "switch.if.OutDiscardPkts", ifstat.IfOutDiscardPkts, ifNameTag, ifIndexTag, ipTag))
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
	RunSnmpRetry(ip, community, timeout, ch, retry, ifNameOid)
}

func IfInDiscardPkts(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifInDiscardpktsOid+task.Ifindex)
}

func IfOutDiscardPkts(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifOutDiscardpktsOid+task.Ifindex)
}

func IfHCInOctets(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifHCInOid+task.Ifindex)
}

func IfHCOutOctets(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifHCOutOid+task.Ifindex)
}

func IfHCInUcastPkts(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifHCInPktsOid+task.Ifindex)
}

func IfHCOutUcastPkts(task *SnmpTask, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	RunSnmpRetry(task.Ip, community, timeout, ch, retry, ifHCOutPktsOid+task.Ifindex)
}

func RunSnmpRetry(ip, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int, oid string) {
	method := "walk"
	var snmpPDUs []gosnmp.SnmpPDU

	for i := 0; i < retry; i++ {
		snmpPDUs, _ = sw.RunSnmp(ip, community, oid, method, timeout)
		if len(snmpPDUs) > 0 {
			ch <- snmpPDUs
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	ch <- snmpPDUs
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

func coreHandleSnmpTaskWalk(task *SnmpTask) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Println(task.Ip+" Recovered in coreHandleSnmpTaskWalk", r)
		}
	}()

	chIfInMap := make(chan map[string]string)
	chIfOutMap := make(chan map[string]string)
	go WalkIfIn(task, community, snmpTimeout, chIfInMap, snmpRetry)
	go WalkIfOut(task, community, snmpTimeout, chIfOutMap, snmpRetry)

	ifInMap := <-chIfInMap
	ifOutMap := <-chIfOutMap
	now := time.Now().Unix()

	chIfInPktMap := make(chan map[string]string)
	chIfOutPktMap := make(chan map[string]string)

	var ifInPktMap, ifOutPktMap map[string]string
	chIfInDcPktMap := make(chan map[string]string)
	chIfOutDcPktMap := make(chan map[string]string)

	var ifInDcPktMap, ifOutDcPktMap map[string]string

	pktcol := checkIgnorePkt()

	if pktcol {
		go WalkIfInPkts(task, community, snmpTimeout, chIfInPktMap, snmpRetry)
		go WalkIfOutPkts(task, community, snmpTimeout, chIfOutPktMap, snmpRetry)
		ifInPktMap = <-chIfInPktMap
		ifOutPktMap = <-chIfOutPktMap
		go WalkIfInDiscardPkts(task, community, snmpTimeout, chIfInDcPktMap, snmpRetry)
		go WalkIfOutDiscardPkts(task, community, snmpTimeout, chIfOutDcPktMap, snmpRetry)
		ifInDcPktMap = <-chIfInDcPktMap
		ifOutDcPktMap = <-chIfOutDcPktMap

	}
	if len(ifInMap) > 0 && len(ifOutMap) > 0 {
		ifstat := &IfStats{}
		ifstat.IfName = task.Ifname
		ifstat.IfIndex = task.Ifindex
		ifstat.IfHCInOctets, _ = strconv.ParseUint(ifInMap[task.Ifindex], 10, 64)
		ifstat.IfHCOutOctets, _ = strconv.ParseUint(ifOutMap[task.Ifindex], 10, 64)

		if pktcol {
			ifstat.IfHCInUcastPkts, _ = strconv.ParseUint(ifInPktMap[task.Ifindex], 10, 64)
			ifstat.IfHCOutUcastPkts, _ = strconv.ParseUint(ifOutPktMap[task.Ifindex], 10, 64)

			ifstat.IfInDiscardPkts, _ = strconv.ParseUint(ifInDcPktMap[task.Ifindex], 10, 32)
			ifstat.IfOutDiscardPkts, _ = strconv.ParseUint(ifOutDcPktMap[task.Ifindex], 10, 32)
		}
		ifstat.TS = now
		issucess := Push2SendQueue(task, ifstat)
		if !issucess && isdebug {
			log.Println(task.Ip + " push IfStatsQueue false!")
		}

		return issucess
	} else {
		log.Println(task.Ip + " get IfStatsQueue false len = 0!")
		return false
	}

}

func WalkIfName(ip, community string, timeout int, ch chan map[string]string, retry int) {
	WalkIf(ip, ifNameOid, community, timeout, retry, ch)
}

func WalkIfIn(task *SnmpTask, community string, timeout int, ch chan map[string]string, retry int) {
	WalkIf(task.Ip, ifHCInOid+task.Ifindex, community, timeout, retry, ch)
}

func WalkIfOut(task *SnmpTask, community string, timeout int, ch chan map[string]string, retry int) {
	WalkIf(task.Ip, ifHCOutOid+task.Ifindex, community, timeout, retry, ch)
}

func WalkIfInPkts(task *SnmpTask, community string, timeout int, ch chan map[string]string, retry int) {
	WalkIf(task.Ip, ifHCInPktsOid+task.Ifindex, community, timeout, retry, ch)
}

func WalkIfOutPkts(task *SnmpTask, community string, timeout int, ch chan map[string]string, retry int) {
	WalkIf(task.Ip, ifHCOutPktsOid+task.Ifindex, community, timeout, retry, ch)
}

func WalkIfInDiscardPkts(task *SnmpTask, community string, timeout int, ch chan map[string]string, retry int) {
	WalkIf(task.Ip, ifInDiscardpktsOid+task.Ifindex, community, timeout, retry, ch)
}

func WalkIfOutDiscardPkts(task *SnmpTask, community string, timeout int, ch chan map[string]string, retry int) {
	WalkIf(task.Ip, ifOutDiscardpktsOid+task.Ifindex, community, timeout, retry, ch)
}

func WalkIf(ip, oid, community string, timeout, retry int, ch chan map[string]string) {
	result := make(map[string]string)

	for i := 0; i < retry; i++ {
		out, err := CmdTimeout(timeout, "snmpwalk", "-v", "2c", "-c", community, ip, oid)
		if err != nil && isdebug {
			log.Println("WalkIf Err", ip, oid, err)
		}

		var list []string
		if strings.Contains(out, "IF-MIB") {
			list = strings.Split(out, "IF-MIB")
		} else {
			list = strings.Split(out, "iso")
		}

		for _, v := range list {

			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered in WalkIf", r)
				}
			}()

			if len(v) > 0 && strings.Contains(v, "=") {
				vt := strings.Split(v, "=")

				var ifIndex, ifName string
				if strings.Contains(vt[0], ".") {
					leftList := strings.Split(vt[0], ".")
					ifIndex = leftList[len(leftList)-1]
					ifIndex = strings.TrimSpace(ifIndex)
				}

				if strings.Contains(vt[1], ":") {
					ifName = strings.Split(vt[1], ":")[1]
					ifName = strings.TrimSpace(ifName)
				}

				result[ifIndex] = ifName
			}
		}

		if len(result) > 0 {
			ch <- result
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	ch <- result
	return
}

func CmdTimeout(timeout int, name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)

	var out bytes.Buffer
	cmd.Stdout = &out

	cmd.Start()
	timer := time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		err := cmd.Process.Kill()
		if err != nil {
			log.Println("failed to kill: ", err)
		}
	})
	err := cmd.Wait()
	timer.Stop()

	return out.String(), err
}
