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
	interval = time.Duration(conf.Transfer.Interval) * time.Second
	isdebug = conf.Debug
	limitCh = make(chan bool, conf.Switch.LimitConcur)

	allSwitchIps = AllSwitchIP()
	lsema = nsema.NewSemaphore(conf.Switch.LimitConcur)
	enableRate = conf.Rate
}

func contains(slice []string, item string) bool {
	if len(slice) == 0 {
		return false
	}
	sort.Strings(slice)
	i := sort.Search(len(slice),
		func(i int) bool { return slice[i] >= item })
	if i < len(slice) && slice[i] == item {
		return true
	}
	return false
}

func AllSwitchIP() (allIP []string) {
	switchIP := g.Config().Switch.IpRange

	if len(switchIP) > 0 {
		for _, sip := range switchIP {
			aip := sw.ParseIp(sip)
			for _, ip := range aip {
				allIP = append(allIP, ip)
			}
		}
	}
	if len(localSwList) > 0 {
		for _, sip := range localSwList {
			if !contains(allIP, sip) {
				allIP = append(allIP, sip)
			}
		}
	}
	return allIP
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
		if IsCollector.IsSet() {
			if len(allSwitchIps) > 0 {
				for _, ip := range allSwitchIps {
					limitCh <- true
					go coreSwIfMap(ip, limitCh)
					time.Sleep(5 * time.Millisecond)
				}

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

func MapEq(olds, news map[string]string) (ischange bool, all map[string]string) {
	ischange = false
	all = make(map[string]string)
	for k, v := range news {
		all[k] = v
	}
	if len(olds) != len(all) {
		ischange = true
	}

	for k, v := range olds {
		w, ok := all[k]
		if !ok {
			all[k] = v
			ischange = true
		}
		if v != w {
			ischange = true
		}
	}

	return ischange, all
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
		ifslist := make(map[string]string)
		//列举交换机snmp oid,ifindex与 ifname 的对应关系
		for _, ifNamePDU := range ifNameList {
			ifName := ifNamePDU.Value.(string)
			isignore := false
			for _, ignore := range ignoreIface {
				if strings.HasPrefix(ifName, ignore) {
					isignore = true
					break
				}
			}
			if isignore {
				continue
			}
			ifIndex := strings.Replace(ifNamePDU.Name, ifNameOidPrefix, "", 1)
			ifslist[ifIndex] = ifName
		}

		//构造任务
		st := GetTaskMap(ip)
		if st == nil {
			//新增任务
			st = new(SnmpTask)
			st.IP = ip
			st.Interval = interval

			AddTask(st)
			st.SetIfsList(ifslist)
			log.Printf("add new st:%v\n", st)
			st.Run()
		} else {
			//对比端口
			oldifs := st.GetIfsList()
			//TODO: 分辨交换机变革,修改对应端口对.(考虑每次 snmpwalk 结果不全.)更新对应 task 的 ifslist.
			isChange, newifs := MapEq(oldifs, ifslist)
			if isChange {
				st.SetIfsList(newifs)
				log.Printf("Find Switch Port Change:%v,%v\n", ip, ifslist)
			}
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
			success := g.SendToTransfer(mvsend)
			pfc.Meter("SWCLSwSend", int64(len(mvsend)))
			log.Println("INFO : Send metrics to transfer running in the background. Send IfStats metrics :", len(mvsend))
			if !success {
				pfc.Meter("SWCLSwSendFails", int64(len(mvsend)))
				log.Println("INFO : Send metrics to transfer running in the background. failed:", len(mvsend))
			}
		}(mvs)
	}

}

func StartSWcollectTask() {

	for _, ip := range allSwitchIps {
		limitCh <- true
		go coreSwIfMap(ip, limitCh)
		time.Sleep(5 * time.Millisecond)
	}

	log.Println("BeginIfstatcollet  ", len(allSwitchIps))
}
