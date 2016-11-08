package lansw

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	pfc "github.com/baishancloud/goperfcounter"
	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/gaochao1/gosnmp"
	"github.com/gaochao1/sw"
	"github.com/getsentry/raven-go"
	"github.com/open-falcon/common/model"
	nsema "github.com/toolkits/concurrent/semaphore"
	nlist "github.com/toolkits/container/list"
)

const (
	DefaultSendQueueMaxSize      = 102400          //10.24w
	DefaultSendTaskSleepInterval = time.Second * 1 //默认睡眠间隔为1s
)

var (
	IfstatsQueue *nlist.SafeListLimited

	//AliveIp []string
	allSwitchIps []string

	pingTimeout int
	pingRetry   int

	community   string
	snmpTimeout int
	snmpRetry   int

	ignoreIface []string
	hightIface  []string
	ignorePkt   bool
	interval    time.Duration
	isdebug     bool
	limitCh     chan bool
	lsema       *nsema.Semaphore
	enableRate  bool
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
	conf := g.Config()
	pingTimeout = conf.Switch.PingTimeout
	pingRetry = conf.Switch.PingRetry

	community = conf.Switch.Community
	snmpTimeout = conf.Switch.SnmpTimeout
	snmpRetry = conf.Switch.SnmpRetry

	ignoreIface = conf.Switch.IgnoreIface
	ignorePkt = conf.Switch.IgnorePkt
	hightIface = conf.Switch.HightIface
	interval = time.Duration(int64(conf.Transfer.Interval)/2) * time.Second
	isdebug = conf.Debug
	limitCh = make(chan bool, conf.Switch.LimitConcur)

	chworkerLm = make(chan bool, 10)
	allSwitchIps = AllSwitchIp()
	lsema = nsema.NewSemaphore(conf.Switch.LimitConcur)
	enableRate = conf.Rate
}

func contains(slice []string, item string) bool {
	sort.Strings(slice)
	i := sort.Search(len(slice),
		func(i int) bool { return slice[i] >= item })
	if i < len(slice) && slice[i] == item {
		return true
	}
	return false
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
	if len(localSwList) > 0 {
		for _, sip := range localSwList {
			if !contains(allIp, sip) {
				allIp = append(allIp, sip)
			}
		}
	}
	return allIp
}

func InitIfstatsQ() {
	IfstatsQueue = nlist.NewSafeListLimited(DefaultSendQueueMaxSize)
}
func StartLanSWCollector() {
	if !g.Config().Switch.Enabled {
		return
	}
	initVariable()
	InitIfstatsQ()
	InitTask()

	InitLanIps()
	go SwifMapChecker()
	SWMetricToTransfer()
}

func SwifMapChecker() {
	log.Println("start SwifMapChecker!")

	for {

		if IsCollector {

			if len(allSwitchIps) > 0 {
				for _, ip := range allSwitchIps {
					limitCh <- true
					go coreSwIfMap(ip, limitCh)
					time.Sleep(5 * time.Millisecond)
				}

			}

			log.Println("Mapchecker end ", len(HightQueue), len(LowQueue), len(chworkerLm), len(allSwitchIps))

			if len(chworkerLm) < 2 {
				go CollectWorker()

			}
		}

		time.Sleep(10 * time.Minute)
	}

}

func pingCheck(ip string) bool {
	var pingResult bool
	for i := 0; i < pingRetry; i++ {
		pingResult = sw.Ping(ip, pingTimeout, true)
		if pingResult == true {
			break
		}
	}
	return pingResult
}

func coreSwIfMap(ip string, limitch chan bool) {
	pingResult := pingCheck(ip)

	if !pingResult {
		<-limitch
		return
	}

	MapIfName(ip)

	<-limitch
}

func MapIfName(ip string) {
	log.Println("MapIfName", ip)
	defer func() {
		if r := recover(); r != nil {
			raven.CaptureError(fmt.Errorf("Recovered in ListIfStats %s , %v", ip, r), nil)
			log.Println("Recovered in ListIfStats", r)
		}
	}()
	chIfNameList := make(chan []gosnmp.SnmpPDU)

	go ListIfName(ip, community, snmpTimeout, chIfNameList, snmpRetry)

	ifNameList := <-chIfNameList
	if len(ifNameList) > 0 {

		for _, ifNamePDU := range ifNameList {

			ifName := ifNamePDU.Value.(string)

			key := fmt.Sprintf("%s/%s", ip, ifName)
			check := !CheckPortMap(key)
			if !check {
				continue
			}
			if len(ignoreIface) > 0 {
				for _, ignore := range ignoreIface {
					if strings.Contains(ifName, ignore) {
						check = false
						break
					}
				}
			}
			if !check {
				continue
			}

			ifIndexStr := strings.Replace(ifNamePDU.Name, ifNameOidPrefix, "", 1)

			it := &SnmpTask{
				Key:       key,
				Ip:        ip,
				Ifindex:   ifIndexStr,
				Ifname:    ifName,
				NextTime:  time.Now(),
				Interval:  interval,
				IgnorePkt: ignorePkt,
			}
			it.PktsNextTime = it.NextTime
			it.IsPriority = false
			for _, hiface := range hightIface {
				if strings.Contains(ifName, hiface) {
					HightQueue <- it
					it.IsPriority = true
					break
				}
			}
			if it.IsPriority == false {
				it.Interval = 2 * it.Interval
				LowQueue <- it
			}

			var im PortMapItem

			im = PortMapItem{
				Key:      key,
				ExpireAt: it.NextTime.Add(it.Interval),
				Ip:       it.Ip,
				Ifname:   it.Ifname,
				Task:     it,
			}

			AddPortMap(key, &im)

		}
	}
}

func SWMetricToTransfer() {
	sema := nsema.NewSemaphore(10)

	for {
		items := IfstatsQueue.PopBackBy(5000)

		count := len(items)
		if count == 0 {
			time.Sleep(DefaultSendTaskSleepInterval)
			continue
		}

		mvs := make([]*model.MetricValue, count)
		for i := 0; i < count; i++ {
			mvs[i] = items[i].(*model.MetricValue)
		}

		//	同步Call + 有限并发 进行发送
		sema.Acquire()
		go func(mvsend []*model.MetricValue) {
			defer sema.Release()
			g.SendToTransfer(mvsend)
			pfc.Meter("SWCLSwSend", int64(len(mvsend)))
			log.Println("INFO : Send metrics to transfer running in the background. Send IfStats metrics :", len(mvsend))
		}(mvs)
	}

}

func StartLanSWcollect() {

	if len(chworkerLm) < 1 {
		go CollectWorker()
	}

	for _, ip := range allSwitchIps {
		limitCh <- true
		go coreSwIfMap(ip, limitCh)
		time.Sleep(5 * time.Millisecond)
	}

	log.Println("BeginIfstatcollet  ", len(HightQueue), len(LowQueue), len(chworkerLm), len(allSwitchIps))
}
func StopLanSWCollect() {
	TurnOffPortMap()
}
