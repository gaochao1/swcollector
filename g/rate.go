package g

import (
	"errors"
	"fmt"
	"sync"
)

var (
	mapCounter map[string]uint64
	mapTime    map[string]int64
	rateLock   = new(sync.RWMutex)
	InterTime  int64
)

func InitCounterMap() {
	mapCounter = map[string]uint64{}
	mapTime = map[string]int64{}
	InterTime = int64(Config().Transfer.Interval)

}
func AlignTs(ts int64, period int64) int64 {
	return ts - ts%period
}

func Rate(ip string, port string, item string, counter uint64, ctime int64) (rate float64, ts []int64, err error) {
	if counter == 0 {
		return 0, nil, errors.New("counter is zero!s")
	}
	key := fmt.Sprintf("%s/%s/%s", ip, port, item)
	rateLock.RLock()
	pcounter, pok := mapCounter[key]
	ptime, tok := mapTime[key]
	rateLock.RUnlock()

	rateLock.Lock()
	mapCounter[key] = counter
	mapTime[key] = ctime
	rateLock.Unlock()

	if !pok || !tok || ctime-ptime > 1800 || ctime-ptime <= 0 {
		return 0, nil, errors.New("not previous record")
	}
	rateval := float64(counter-pcounter) / float64(ctime-ptime)
	if rateval > 1442177280 || rateval < 0 {
		err := errors.New(fmt.Sprintf("calc rate error. at time:%v,  current:%v,pretime:%v,previous:%v", ctime, counter, ptime, pcounter))
		return 0, nil, err
	}
	pats := AlignTs(ptime, InterTime)
	ats := AlignTs(ctime, InterTime)
	if (ats-pats)/InterTime == 1 {
		ats := AlignTs(ctime, InterTime)
		return rateval, []int64{ats}, nil
	}

	ts = []int64{}
	for t := pats + InterTime; t <= ats; t += InterTime {
		ts = append(ts, t)
	}
	return rateval, ts, nil
}
