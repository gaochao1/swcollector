package ifstat

import (
	"log"
	"strings"
	"time"

	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/gaochao1/gosnmp"
	"github.com/gaochao1/sw"
	"github.com/open-falcon/common/model"
	nsema "github.com/toolkits/concurrent/semaphore"
	nlist "github.com/toolkits/container/list"
	"github.com/toolkits/slice"
)

const (
	DefaultSendQueueMaxSize      = 102400                //10.24w
	DefaultSendTaskSleepInterval = time.Millisecond * 20 //默认睡眠间隔为50ms
)

var (
	IfstatsQueue *nlist.SafeListLimited

	AliveIp []string
	AllIp   []string

	pingTimeout int
	pingRetry   int

	community   string
	snmpTimeout int
	snmpRetry   int

	ignoreIface []string
	ignorePkt   bool
	interval    time.Duration
	isdebug     bool
)

type IfStats struct {
	IfName           string
	IfIndex          string
	IfHCInOctets     uint64
	IfHCOutOctets    uint64
	IfHCInUcastPkts  uint64
	IfHCOutUcastPkts uint64
	IfInDiscardPkts  uint64
	IfOutDiscardPkts uint64
	TS               int64
}

func initVariable() {
	pingTimeout = g.Config().Switch.PingTimeout
	pingRetry = g.Config().Switch.PingRetry

	community = g.Config().Switch.Community
	snmpTimeout = g.Config().Switch.SnmpTimeout
	snmpRetry = g.Config().Switch.SnmpRetry

	ignoreIface = g.Config().Switch.IgnoreIface
	ignorePkt = g.Config().Switch.IgnorePkt
	interval = time.Duration(int64(g.Config().Transfer.Interval)) * time.Second
	isdebug = g.Config().Debug

	chworkerLm = make(chan bool, g.Config().Switch.LimitConcur)

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

func InitIfstatsQ() {
	IfstatsQueue = nlist.NewSafeListLimited(DefaultSendQueueMaxSize)
}
func StartIfStatsCollector() {
	if !g.Config().Switch.Enabled || len(g.Config().Switch.IpRange) == 0 {
		return
	}
	initVariable()
	InitIfstatsQ()
	InitTask()

	go SwifMapChecker()
	SWifMetricToTransfer()
}

func SwifMapChecker() {
	log.Println("start SwifMapChecker!")
	AllIp = AllSwitchIp()
	for i := 0; i < len(AllIp); i++ {
		go CollectWorker()
	}
	limitCh := make(chan bool, g.Config().Switch.LimitConcur)
	for {
		for _, ip := range AllIp {
			limitCh <- true
			go coreSwIfMap(ip, limitCh)
			time.Sleep(5 * time.Millisecond)
		}
		log.Println("Mapchecker end ", len(HightQueue), len(LowQueue), len(chworkerLm), len(AllIp))
		lt := <-LowQueue
		LowQueue <- lt
		if lt.NextTime.Before(time.Now()) && len(chworkerLm) < MaxTaskWorkers-10 {
			go CollectWorker()

		}

		time.Sleep(10 * time.Minute)
	}

}

func pingCheck(ip string) bool {
	var pingResult bool
	for i := 0; i < pingRetry; i++ {
		pingResult = sw.Ping(ip, pingTimeout)
		if pingResult == true {
			break
		}
	}
	return pingResult
}

func coreSwIfMap(ip string, limitch chan bool) {
	pingResult := pingCheck(ip)
	snmpWalk := false

	if !pingResult {
		DeletePortMap(ip)
		<-limitch
		return
	}
	vendor, err := sw.SysVendor(ip, community, snmpTimeout)
	if err != nil {
		return
	}
	if !slice.ContainsString(AliveIp, ip) {
		AliveIp = append(AliveIp, ip)
	}
	snmpWalk = true
	log.Println("port map add ", ip, vendor)
	if !snmpWalk {
		MapIfName(ip)
	} else {
		MapIfNameWalk(ip)
	}

}
func MapIfNameWalk(ip string) {
	log.Println("MapIfNameWalk", ip)
	chIfNameMap := make(chan map[string]string)

	go WalkIfName(ip, community, snmpTimeout, chIfNameMap, snmpRetry)
	ifNameMap := <-chIfNameMap

	if len(ifNameMap) > 0 {
		for ifIndex, ifName := range ifNameMap {

			check := !CheckPortMap(ip, ifName)
			if len(ignoreIface) > 0 {
				for _, ignore := range ignoreIface {
					if strings.Contains(ifName, ignore) {
						check = false
						break
					}
				}
			}

			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered in ListIfStats_SnmpWalk", r)
				}
			}()
			if check {

				it := &SnmpTask{ip, ifIndex, ifName, true, time.Now(), interval, Flow, ignorePkt}
				if strings.Contains(ifName, "XGigabitEthernet") || strings.Contains(ifName, "Te0") {
					HightQueue <- it
				} else {
					LowQueue <- it
				}
				var im PortMapItem

				im = PortMapItem{it.NextTime, it.NextTime, false}
				AddPortMap(ip, ifName, &im)

			}

		}
		log.Println("MapWalk end ", ip, len(HightQueue), len(LowQueue), len(GPortMap))
	} else {
		log.Println("snmpWalk get ifname false", ip)
	}
}

func MapIfName(ip string) {
	log.Println("MapIfName", ip)
	chIfNameList := make(chan []gosnmp.SnmpPDU)

	go ListIfName(ip, community, snmpTimeout, chIfNameList, snmpRetry)

	ifNameList := <-chIfNameList
	if len(ifNameList) > 0 {

		for _, ifNamePDU := range ifNameList {

			ifName := ifNamePDU.Value.(string)

			check := !CheckPortMap(ip, ifName)

			if len(ignoreIface) > 0 && check {
				for _, ignore := range ignoreIface {
					if strings.Contains(ifName, ignore) {
						check = false
						break
					}
				}
			}

			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered in ListIfStats", r)
				}
			}()

			if check {
				ifIndexStr := strings.Replace(ifNamePDU.Name, ifNameOidPrefix, "", 1)
				it := &SnmpTask{ip, ifIndexStr, ifName, false, time.Now(), interval, Flow, ignorePkt}
				if strings.Contains(ifName, "XGigabitEthernet") || strings.Contains(ifName, "Te0") {
					HightQueue <- it
				} else {
					LowQueue <- it
				}
				var im PortMapItem

				im = PortMapItem{it.NextTime, it.NextTime, false}

				AddPortMap(ip, ifName, &im)

			}
		}
	}
}

func SWifMetricToTransfer() {
	log.Println("start SWifMetricToTransfer")
	sema := nsema.NewSemaphore(10)

	for {
		items := IfstatsQueue.PopBackBy(5000)

		count := len(items)
		if count == 0 {
			time.Sleep(DefaultSendTaskSleepInterval)
			continue
		}

		mvsSend := make([]*model.MetricValue, count)
		for i := 0; i < count; i++ {
			mvsSend[i] = items[i].(*model.MetricValue)
		}

		//	同步Call + 有限并发 进行发送
		sema.Acquire()
		go func(mvsend []*model.MetricValue) {
			defer sema.Release()
			g.SendToTransfer(mvsend)
		}(mvsSend)
	}

}
