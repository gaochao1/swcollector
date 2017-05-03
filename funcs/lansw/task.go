package lansw

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	pfc "github.com/baishancloud/goperfcounter"
)

var (
	TasksMap map[string]*SnmpTask
	tmlock   = new(sync.RWMutex)
)

type IfOids struct {
	IfHCOutIn     []string
	IfHCPktsOutIn []string
}

type SnmpTask struct {
	IP       string
	IfsList  map[string]string
	IfsOids  map[string]IfOids
	Interval time.Duration
	runflag  int32
	sync.RWMutex
}

func (st *SnmpTask) GetIfsList() map[string]string {
	st.RLock()
	defer st.RUnlock()
	return st.IfsList
}

func (st *SnmpTask) GetIfsOids() map[string]IfOids {
	st.RLock()
	defer st.RUnlock()
	return st.IfsOids
}

func (st *SnmpTask) SetIfsList(ifs map[string]string) {
	st.Lock()
	defer st.Unlock()
	st.IfsList = ifs
	oidsmap := make(map[string]IfOids)

	for index, name := range ifs {
		oids := IfOids{
			IfHCOutIn:     []string{ifHCOutOid + index, ifHCInOid + index},
			IfHCPktsOutIn: []string{ifHCOutPktsOid + index, ifHCInPktsOid + index, ifOutDiscardpktsOid + index, ifInDiscardpktsOid + index},
		}
		oidsmap[name] = oids
	}
	st.IfsOids = oidsmap

}
func (st *SnmpTask) SetRun() bool {
	return atomic.CompareAndSwapInt32(&(st.runflag), 0, 1)
}

func (st *SnmpTask) Stop() {
	atomic.StoreInt32(&(st.runflag), 0)
}

func (st *SnmpTask) Run() {
	ticker := time.NewTicker(st.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if !IsCollector.IsSet() {
				continue
			}
			if st.SetRun() {
				go func() {
					result := handleSnmpTask(st)
					if result == false {
						log.Println("SnmpTask false Time", st.IP)
						pfc.Report("SWitchSNMPFalse,ip="+st.IP, "switch("+st.IP+")SnmpGetAllFalse")
					}
					st.Stop()
				}()

			} else {
				log.Println("SnmpTask Run Expired!", st.IP)
				pfc.Report("SnmpSnmpExpired,ip="+st.IP, "switch("+st.IP+")SnmpGetExpired")
			}
		}
	}

}

func GetTaskMap(ip string) *SnmpTask {
	tmlock.RLock()
	defer tmlock.RUnlock()
	return TasksMap[ip]
}

func AddTask(st *SnmpTask) {
	tmlock.Lock()
	defer tmlock.Unlock()
	TasksMap[st.IP] = st
}

func InitTask() {
	TasksMap = make(map[string]*SnmpTask)
}
