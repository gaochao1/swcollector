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
)

func InitCounterMap() {
	mapCounter = map[string]uint64{}
	mapTime = map[string]int64{}

}

func Rate(ip string, port string, item string, counter uint64, ctime int64) (float64, error) {
	if counter == 0 {
		return 0, errors.New("counter is zero!s")
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
		return 0, errors.New("not previous record")
	}
	rate := float64(counter-pcounter) / float64(ctime-ptime)
	if rate > 1250000000 || rate < 0 {
		err := errors.New(fmt.Sprintf("calc rate error. at time:%v,  current:%v,pretime:%v,previous:%v", ctime, counter, ptime, pcounter))
		return 0, err
	}

	return rate, nil
}
