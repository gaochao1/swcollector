package lansw

import (
	"log"
	"sync"
	"time"
)

const (
	TaskDefaultNums = 200
	MaxTaskWorkers  = 2000
)

//const (
//	Flow = iota
//	Pkts
//)

var (
	HightQueue chan *SnmpTask
	LowQueue   chan *SnmpTask
	lock       = new(sync.RWMutex)
	chworkerLm chan bool
	GPortMap   map[string]*PortMapItem
)

func AddPortMap(key string, im *PortMapItem) {
	lock.Lock()
	GPortMap[key] = im
	lock.Unlock()
}
func DeletePortMap(key string) {
	lock.Lock()
	delete(GPortMap, key)
	lock.Unlock()
}

func CheckPortMapTask(key string, t *SnmpTask) bool {
	lock.RLock()
	im, ex := GPortMap[key]
	lock.RUnlock()
	if ex {
		if im.Task == t {
			return true
		}
		return time.Now().After(im.ExpireAt)
	}
	return false
}
func UpdatePortMapTime(key string) {
	lock.RLock()
	im, ex := GPortMap[key]
	lock.RUnlock()
	if ex && im.Task != nil {
		im.ExpireAt = im.Task.NextTime.Add(im.Task.Interval)
	}
}

func TurnOffPortMap() {
	ext := time.Now().Add(interval * time.Duration(3))
	lock.RLock()
	for _, i := range GPortMap {
		i.ExpireAt = ext
	}
	lock.RUnlock()
}

func CheckPortMap(key string) bool {
	lock.RLock()
	im, ex := GPortMap[key]
	lock.RUnlock()
	if ex {
		return time.Now().Before(im.ExpireAt)
	}
	return false
}

type SnmpTask struct {
	Key          string
	Ip           string
	Ifindex      string
	Ifname       string
	Interval     time.Duration
	CollectType  int
	IgnorePkt    bool
	IsPriority   bool
	FalseTimes   int
	NextTime     time.Time
	PktsNextTime time.Time
}
type PortMapItem struct {
	Key      string
	ExpireAt time.Time
	Ip       string
	Ifname   string
	Task     *SnmpTask
}

func InitTask() {
	HightQueue = make(chan *SnmpTask, TaskDefaultNums)
	LowQueue = make(chan *SnmpTask, TaskDefaultNums*10)
	GPortMap = make(map[string]*PortMapItem)

}

func CollectWorker() {
	chworkerLm <- true
	defer func() {
		<-chworkerLm
		if isdebug {
			log.Println("CollectWorker End")
		}
	}()
	if isdebug {
		log.Println("CollectWorker start")
	}
	i := 0
	var t, tl *SnmpTask
	for {
		if len(HightQueue) > 0 {
			t = <-HightQueue
		}
		if t != nil && t.NextTime.Before(time.Now()) {
			go checkAndRunTask(t)
			t = nil
			i = 0
			continue
		} else {
			if t != nil {
				HightQueue <- t
				t = nil
			}
			if len(LowQueue) > 0 {
				tl = <-LowQueue
			}
			if tl != nil && tl.NextTime.Before(time.Now()) {
				go checkAndRunTask(tl)
				tl = nil
				i = 0
				continue
			} else {
				if tl != nil {
					LowQueue <- tl
					tl = nil
				}
				i++
				if i > (len(HightQueue)+len(LowQueue))/len(chworkerLm) {
					if isdebug {
						log.Println("Sleep,", len(HightQueue), len(LowQueue), len(chworkerLm))
					}
					time.Sleep(time.Second)
				}
			}
		}

	}
}

func checkAndRunTask(t *SnmpTask) {
	if !CheckPortMapTask(t.Key, t) {
		log.Println("cant run Task", t.Ip, t.Ifname)
		return
	}
	if IsCollector {
		if !t.IsPriority {
			lsema.Acquire()
			defer func() {
				if !t.IsPriority {
					lsema.Release()
				}
			}()
		}
		result := handleSnmpTask(t)
		if result == false {
			t.FalseTimes++
			if t.FalseTimes > 13 {
				DeletePortMap(t.Key)
				return
			}
		} else {
			t.FalseTimes = 0
		}
		if isdebug {
			log.Println(t.Ip, t.Ifname, "run result:", result, t.FalseTimes, t.Interval)
		}
	} else {
		log.Println("im not collector ,cant run Task", t.Ip, t.Ifname)
	}
	dt := t.Interval - time.Second
	if !t.IsPriority && t.FalseTimes > 2 {
		dt = time.Duration(t.FalseTimes-2)*t.Interval - time.Second
	}
	t.NextTime = time.Now().Add(dt)
	UpdatePortMapTime(t.Key)

	if t.IsPriority {
		HightQueue <- t
	} else {
		LowQueue <- t
	}

}
