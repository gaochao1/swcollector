package ifstat

import (
	"log"
	"sync"
	"time"
)

const (
	TaskDefaultNums = 200
	MaxTaskWorkers  = 2000
)
const (
	Flow = iota
	Pkts
)

var (
	HightQueue chan *SnmpTask
	LowQueue   chan *SnmpTask
	lock       = new(sync.RWMutex)
	chworkerLm chan bool
	GPortMap   map[string]map[string]*PortMapItem
)

func AddPortMap(ip, port string, im *PortMapItem) {
	if ipm, ex := GPortMap[ip]; ex {
		ipm[port] = im
	} else {
		lock.RLock()
		ipm := make(map[string]*PortMapItem)
		GPortMap[ip] = ipm
		lock.RUnlock()
		ipm[port] = im

	}
}
func DeletePortMap(ip string) {
	lock.RLock()
	//defer lock.RUnlock()
	delete(GPortMap, ip)
	lock.RUnlock()
}
func DeletePortMapPort(ip, port string) {
	lock.RLock()
	//defer lock.RUnlock()
	//delete(GPortMap, ip)
	if ipm, ex := GPortMap[ip]; ex {
		delete(ipm, port)
	} else {
		delete(GPortMap, ip)
	}

	lock.RUnlock()
}

func CheckPortMap(ip, port string) bool {
	if ipm, ex := GPortMap[ip]; ex {
		if im, ex := ipm[port]; ex {
			if im.NextTime.Add(interval * 5).After(time.Now()) {
				return true
			}
		}
	}
	return false
}
func CheckPortMapTime(ip, port string, tm time.Time) bool {
	if ipm, ex := GPortMap[ip]; ex {
		if im, ex := ipm[port]; ex {
			if im.NextTime == tm {
				return true
			}
		}
	}
	return false
}
func CheckPortMapIP(ip string) bool {
	if ipm, ex := GPortMap[ip]; ex {
		for _, v := range ipm {
			if v.NextTime.Add(interval * 6).After(time.Now()) {
				return true
			}
		}
	}
	return false
}
func UpdatePortMapTime(ip, port string, tm time.Time) bool {
	if ipm, ex := GPortMap[ip]; ex {
		if im, ex := ipm[port]; ex {
			im.NextTime = tm
			return true
		}
	}
	return false
}

type SnmpTask struct {
	Ip          string
	Ifindex     string
	Ifname      string
	SnmpWalk    bool
	NextTime    time.Time
	Interval    time.Duration
	CollectType int
	IgnorePkt   bool
}
type PortMapItem struct {
	NextTime     time.Time
	PktsNextTime time.Time
	IsRuned      bool
	//IsReach      bool
}

func InitTask() {
	HightQueue = make(chan *SnmpTask, TaskDefaultNums)
	LowQueue = make(chan *SnmpTask, TaskDefaultNums*10)
	GPortMap = make(map[string]map[string]*PortMapItem)

}

func CollectWorker() {
	if isdebug {
		log.Println("CollectWorker start")
	}
	chworkerLm <- true
	defer func() {
		<-chworkerLm

	}()
	i := 0
	for {
		t := <-HightQueue
		if t.NextTime.Before(time.Now()) {
			checkAndRunTask(t, HightQueue)
			i = 0
			continue
		} else {
			HightQueue <- t
			tl := <-LowQueue
			if tl.NextTime.Before(time.Now()) {
				checkAndRunTask(tl, LowQueue)
				i = 0
				continue
			} else {
				LowQueue <- tl
				i++
				if i > (len(HightQueue)+len(LowQueue))/len(chworkerLm)+1 {
					if len(chworkerLm) > len(AllIp) {
						return
					}
					if isdebug {
						log.Println("Sleep,", len(HightQueue), len(LowQueue), len(chworkerLm))
					}
					time.Sleep(time.Second)
				}
			}
		}

	}
}

func checkAndRunTask(t *SnmpTask, cht chan *SnmpTask) {
	runs := handleSnmpTask(t)
	if CheckPortMap(t.Ip, t.Ifname) {
		t.NextTime = time.Now().Add(t.Interval- time.Second)
		if runs {
			UpdatePortMapTime(t.Ip, t.Ifname, t.NextTime)
		}

		cht <- t
	} else {
		DeletePortMapPort(t.Ip, t.Ifname)

	}
}
