package lansw

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	pfc "github.com/baishancloud/goperfcounter"
	"github.com/baishancloud/octopux-swcollector/funcs"
	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/gaochao1/gosnmp"
	"github.com/gaochao1/sw"
	"github.com/getsentry/raven-go"
	"github.com/open-falcon/common/model"
)

const (
	ifNameOid                 = "1.3.6.1.2.1.31.1.1.1.1"
	ifNameOidPrefix           = ".1.3.6.1.2.1.31.1.1.1.1."
	ifHCInOid                 = "1.3.6.1.2.1.31.1.1.1.6."
	ifHCInOidPrefix           = ".1.3.6.1.2.1.31.1.1.1.6."
	ifHCOutOid                = "1.3.6.1.2.1.31.1.1.1.10."
	ifHCOutOidPrefix          = ".1.3.6.1.2.1.31.1.1.1.10."
	ifHCInPktsOid             = "1.3.6.1.2.1.31.1.1.1.7."
	ifHCInPktsOidPrefix       = ".1.3.6.1.2.1.31.1.1.1.7."
	ifHCOutPktsOid            = "1.3.6.1.2.1.31.1.1.1.11."
	ifHCOutPktsOidPrefix      = ".1.3.6.1.2.1.31.1.1.1.11."
	ifInDiscardpktsOid        = "1.3.6.1.2.1.2.2.1.13."
	ifInDiscardpktsOidPrefix  = ".1.3.6.1.2.1.2.2.1.13."
	ifOutDiscardpktsOid       = "1.3.6.1.2.1.2.2.1.19."
	ifOutDiscardpktsOidPrefix = ".1.3.6.1.2.1.2.2.1.19."
)

const (
	INBYTE = iota
	OUTBYTE
	INPKT
	OUTPKT
	INDCPKT
	OUTDCPKT
	UNKNOWN
)

type AsyncPDUResult struct {
	Stamp int64
	PDUs  map[string]uint64
}

func getPduInfo(oid string) (kind int, index string) {

	lastdot := strings.LastIndex(oid, ".")
	if lastdot < 21 || lastdot > 24 {
		return UNKNOWN, ""
	}
	switch oid[0 : lastdot+1] {
	case ifHCInOidPrefix:
		kind = INBYTE
	case ifHCOutOidPrefix:
		kind = OUTBYTE
	case ifHCInPktsOidPrefix:
		kind = INPKT
	case ifHCOutPktsOidPrefix:
		kind = OUTPKT
	case ifInDiscardpktsOidPrefix:
		kind = INDCPKT
	case ifOutDiscardpktsOidPrefix:
		kind = OUTDCPKT
	default:
		kind = UNKNOWN
	}
	return kind, oid[lastdot+1:]
}

func getMetricName(kind int, israte bool) string {
	switch kind {
	case INBYTE:
		if !israte {
			return "switch.if.In"
		}
		return "swinrate"
	case OUTBYTE:
		if !israte {
			return "switch.if.Out"
		}
		return "swoutrate"

	case INPKT:
		if !israte {
			return "switch.if.InPkts"
		}
		return "swinpktsrate"
	case OUTPKT:
		if !israte {
			return "switch.if.OutPkts"
		}
		return "swoutpktsrate"
	case INDCPKT:
		if !israte {
			return "switch.if.InDisCardPkts"
		}
		return "swindcpktsrate"
	case OUTDCPKT:
		if !israte {
			return "switch.if.OutDiscardPkts"
		}
		return "swoutdcpktsrate"
	default:
		return "unknown"
	}
}

func handleSnmpTask(st *SnmpTask) (result bool) {
	defer func() {
		if r := recover(); r != nil {
			raven.CaptureError(fmt.Errorf("Recovered in handleSnmpTask %s , %v", st.IP, r), nil)
			log.Println(st.IP, " Recovered in handleSnmpTask", r)
			result = false
		}
	}()

	pfc.Meter("SWCLSnmpCnt", int64(1))
	result = CollectIfHC(st)
	pktcol := checkPktCollect()
	if pktcol {
		result = CollectIfPkt(st) || result
	}
	return result
}

func CollectIfPkt(st *SnmpTask) bool {
	ifoids := st.GetIfsOids()

	alloids := make([]string, 0)
	for _, oids := range ifoids {
		alloids = append(alloids, oids.IfHCPktsOutIn...)
	}

	return DoAllSnmp(st, alloids)

}

func CollectIfHC(st *SnmpTask) bool {
	ifoids := st.GetIfsOids()

	alloids := make([]string, 0)
	for _, oids := range ifoids {
		alloids = append(alloids, oids.IfHCOutIn...)
	}
	return DoAllSnmp(st, alloids)
}

func DoAllSnmp(st *SnmpTask, oids []string) bool {
	step := 10
	AsyncPDUCh := make(chan *AsyncPDUResult, len(oids)/step+1)
	retCh := make(chan bool)
	go ReadSnmpPDUs(st, len(oids)/step+1, AsyncPDUCh, retCh)
	wg := new(sync.WaitGroup)
	for i := 0; i < len(oids); i += step {
		end := i + step
		if end > len(oids) {
			end = len(oids)
		}
		wg.Add(1)
		go func() {
			SnmpMultiGet(st.IP, community, snmpTimeout, AsyncPDUCh, snmpRetry, oids[i:end])
			wg.Done()
		}()

		time.Sleep(200 * time.Millisecond)
	}

	wg.Wait()
	close(AsyncPDUCh)
	return <-retCh
}

func ReadSnmpPDUs(st *SnmpTask, times int, asyncPDUCh chan *AsyncPDUResult, retCh chan bool) {
	iflist := st.GetIfsList()
	ipTag := "ip=" + st.IP
	falsecnt := 0
	for {
		pdupack, ok := <-asyncPDUCh
		if !ok {
			break
		}
		if len(pdupack.PDUs) < 1 || pdupack.Stamp < 1 {
			falsecnt++
			continue
		}

		var L []*model.MetricValue

		for k, v := range pdupack.PDUs {

			kind, index := getPduInfo(k)

			ifname, ok := iflist[index]
			if !ok {
				log.Printf("ReadSnmpPDU no find ifname %v,%v\n", k, v)
				continue
			}
			metric := getMetricName(kind, enableRate)
			ifNameTag := "ifName=" + ifname
			ifIndexTag := "ifIndex=" + index
			if !enableRate {
				L = append(L, funcs.CounterValueIp(pdupack.Stamp, st.IP, metric, v, ifNameTag, ifIndexTag, ipTag))
			} else {
				rate, ts, err := g.Rate(st.IP, ifname, metric, v, pdupack.Stamp)
				if err == nil {
					L = append(L, funcs.GaugeValueIpSlicTs(ts, st.IP, metric, rate, ifNameTag, ifIndexTag, ipTag)...)
				}
			}
		}

		if len(L) == 0 {
			falsecnt++
			continue
		}

		interfaceSlice := make([]interface{}, len(L))
		for i, d := range L {
			interfaceSlice[i] = d
		}

		if !IfstatsQueue.PushFrontBatch(interfaceSlice) {
			falsecnt++
		}

	}
	retCh <- (falsecnt < times)
}

func SnmpMultiGet(ip, community string, timeout int, ch chan *AsyncPDUResult, retry int, oids []string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in SnmpMultiGet", r)
		}
	}()
	pdus := make(map[string]uint64)
	stamp := int64(0)
	args := []string{"-v", "2c", "-t", strconv.Itoa(timeout / 1000), "-O", "fn", "-c", community, ip}
	args = append(args, oids...)
	for i := 0; i < retry; i++ {
		pfc.Counter("RunSnmp", 1)
		out, err := CmdTimeout(timeout, "snmpget", args...)
		if err != nil {
			pfc.Counter("RunSnmpError", 1)
			log.Println("SnmpMultiGet error", ip, oids, err)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		stamp = time.Now().Unix()
		lines := strings.Split(out, "\n")
		for _, p := range lines {
			if len(p) == 0 {
				continue
			}
			pt := strings.Split(p, "=")
			if len(pt) != 2 {
				log.Println("snmpget return error:", p)
				continue
			}
			k := strings.TrimSpace(pt[0])
			v := pt[1]
			vt := strings.Split(v, ":")
			if len(vt) != 2 {
				log.Println("snmpget return error:", p)
				continue
			}
			vv, err := strconv.ParseUint(strings.TrimSpace(vt[1]), 10, 64)
			if err != nil {
				log.Println("snmpget return error:", p, err)
				continue
			}
			if vv > 0 {
				pdus[k] = vv
			}
		}
		if len(lines) > 0 {
			if len(lines) < len(oids) {
				pfc.Counter("RunSnmpError", 1)
			}
			break
		}
		pfc.Counter("RunSnmpError", 1)
		log.Println("SnmpMultiGet error", ip, oids, err)
		time.Sleep(200 * time.Millisecond)
	}
	apdus := &AsyncPDUResult{
		Stamp: stamp,
		PDUs:  pdus,
	}
	ch <- apdus

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

func ListIfName(ip, community string, timeout int, ch chan []gosnmp.SnmpPDU, retry int) {
	method := "walk"
	var snmpPDUs []gosnmp.SnmpPDU
	var err error
	for i := 0; i < retry; i++ {
		pfc.Counter("RunSnmp", 1)
		snmpPDUs, err = sw.RunSnmp(ip, community, ifNameOid, method, timeout)
		if len(snmpPDUs) > 0 {
			ch <- snmpPDUs
			return
		}
		pfc.Counter("RunSnmpError", 1)
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		pfc.Meter("SWCLSnmpTW,ip="+ip, int64(1))
		log.Println("RunSnmpRetry error:", ip, ifNameOid, method, timeout, err.Error())
	}
	ch <- []gosnmp.SnmpPDU{}
	return
}

func checkPktCollect() bool {
	if ignorePkt {
		return !ignorePkt
	}
	unixtime := time.Now().Unix()
	intervalsec := int64(interval.Seconds())
	return (unixtime/intervalsec)%5 == 0
}
