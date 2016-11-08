package g

import (
	"log"
	"math"
	"math/rand"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/open-falcon/common/model"
)

var (
	TransferClientsLock *sync.RWMutex                   = new(sync.RWMutex)
	TransferClients     map[string]*SingleConnRpcClient = map[string]*SingleConnRpcClient{}
	IPlock              *sync.RWMutex                   = new(sync.RWMutex)
	LookupCount         int                             = 0
	sendips             []string
	FailTimes           []uint32
)

func GetHost() {
	newips := make([]string, 0)
	pos := strings.Index(Config().Transfer.Addr, ":")
	port := Config().Transfer.Addr[pos:]
	temp, err := net.LookupHost(Config().Transfer.Addr[:pos])
	if err != nil {
		if Config().Debug {
			log.Printf("%s Lookup Host Err: %s", Config().Transfer.Addr, err.Error())
		}
	} else {
		for _, j := range temp {
			newips = append(newips, j+port)
		}
	}

	sort.Strings(newips)
	Same := true
	if len(newips) != len(sendips) {
		Same = false
	} else {
		for i, ip := range newips {
			if ip != sendips[i] {
				Same = false
				break
			}
		}
	}

	if Same {
		return
	}

	l := len(newips)
	ft := make([]uint32, l)
	for i := 0; i < l; i++ {
		ft[i] = 0
	}

	IPlock.Lock()
	FailTimes = ft
	sendips = newips
	IPlock.Unlock()
}

func SendMetrics(metrics []*model.MetricValue, resp *model.TransferResponse) {

	rand.Seed(time.Now().UnixNano())

	LookupCount += 1
	if LookupCount >= 1000 {
		LookupCount = 0
		go GetHost()
	}

	ipsNum := len(sendips)

	idx := rand.Int() % ipsNum
	for times := 0; times < ipsNum; times++ {
		addr := sendips[idx]

		if _, ok := TransferClients[addr]; !ok {
			initTransferClient(addr)
		}

		if updateMetrics(addr, metrics, resp) {
			break
		} else {
			FailTimes[idx] += 1
			minTimes := FailTimes[0]
			minID := 0
			for i, f := range FailTimes {
				if i != idx && f < minTimes {
					minTimes = f
					minID = i
				}
			}
			if Config().Debug {
				log.Printf("%s send fail, change to ip %s\n", sendips[idx], sendips[minID])
			}
			idx = minID
			if FailTimes[idx] == math.MaxUint32 {
				for i, _ := range FailTimes {
					FailTimes[i] = 0
				}
			}
		}
	}
}

func initTransferClient(addr string) {
	TransferClientsLock.Lock()
	TransferClients[addr] = &SingleConnRpcClient{
		RpcServer: addr,
		Timeout:   time.Duration(Config().Transfer.Timeout) * time.Millisecond,
	}
	TransferClientsLock.Unlock()
}

func updateMetrics(addr string, metrics []*model.MetricValue, resp *model.TransferResponse) bool {
	TransferClientsLock.RLock()
	client := TransferClients[addr]
	TransferClientsLock.RUnlock()

	err := client.Call("Transfer.Update", metrics, resp)

	if err != nil {
		log.Println("call Transfer.Update fail", addr, err)
		return false
	}
	return true
}
