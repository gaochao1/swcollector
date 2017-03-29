package lansw

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/gaochao1/sw"
	tnet "github.com/toolkits/net"
	"github.com/toolkits/net/httplib"
)

var (
	lanNetIps     = make([]string, 0)
	aliveList     = make([]string, 0)
	myExternalIps = make([]string, 0)
	localSwList   = make([]string, 0)
	IsCollector   bool
)

func ExternalIP() (ips []string, wnets []*net.IPNet) {
	ips = make([]string, 0)
	wnets = make([]*net.IPNet, 0)
	ifaces, e := net.Interfaces()
	if e != nil {
		return
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}

		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}

		// ignore docker and warden bridge
		if strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "w-") {
			continue
		}

		addrs, e := iface.Addrs()
		if e != nil {
			return
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			ipStr := ip.String()
			if !tnet.IsIntranet(ipStr) {
				ips = append(ips, ipStr)
				wnets = append(wnets, addr.(*net.IPNet))
			}
		}
	}
	//fmt.Printf("res:%v\n", ips)
	return
}

func Hosts(ipnet *net.IPNet) ([]string, error) {
	var ips []string
	ip := ipnet.IP.Mask(ipnet.Mask)

	for ; ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	if len(ips) <= 1 {
		return nil, fmt.Errorf("the ipnet is empty %v。\n", ipnet)
	}
	return ips[1 : len(ips)-1], nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

type Pong struct {
	Ip    string
	Alive bool
	IsSw  bool
}

func doPing(pingChan <-chan string, pongChan chan<- Pong) {
	for {
		ip, ok := <-pingChan
		if !ok {
			return
		}
		if contains(allSwitchIps, ip) {
			pongChan <- Pong{Ip: ip, Alive: true, IsSw: true}
			continue
		}
		alive, isSw := false, false
		if !contains(aliveList, ip) {
			_, err := exec.Command("ping", "-c1", "-t6", ip).Output()

			alive = (err == nil)
			if alive == false {
				pongChan <- Pong{Ip: ip, Alive: alive, IsSw: isSw}
				continue
			}
		} else {
			alive = true
		}

		swVendor, _ := sw.SysVendor(ip, community, 1000)
		isSw = (len(swVendor) > 0 && swVendor != "Linux")
		pongChan <- Pong{Ip: ip, Alive: alive, IsSw: isSw}
	}
}

func receivePong(pongNum int, pongChan <-chan Pong, doneChan chan<- []Pong) {
	var alives []Pong
	for i := 0; i < pongNum; i++ {
		pong := <-pongChan
		if pong.Alive {
			alives = append(alives, pong)
		}
	}
	doneChan <- alives
}

func UpdateSwList() {
	concurrentMax := 100
	pingChan := make(chan string, concurrentMax)
	pongChan := make(chan Pong, len(lanNetIps))
	doneChan := make(chan []Pong)

	for i := 0; i < concurrentMax && i < len(lanNetIps); i++ {
		go doPing(pingChan, pongChan)
	}

	cnt := 0
	for _, ip := range lanNetIps {
		pingChan <- ip
		cnt++
	}
	go receivePong(cnt, pongChan, doneChan)
	close(pingChan)

	alives := <-doneChan
	close(pongChan)
	close(doneChan)
	//localSwList = make([]string, 0)
	//aliveList = make([]string, 0)
	for _, p := range alives {
		if p.Alive && !p.IsSw && !contains(aliveList, p.Ip) {
			aliveList = append(aliveList, p.Ip)
		} else if p.Alive && p.IsSw && !contains(localSwList, p.Ip) {
			localSwList = append(localSwList, p.Ip)
		}
	}
	sort.Strings(aliveList)
	sort.Strings(localSwList)
	allSwitchIps = AllSwitchIp()
	log.Println("All switch ips : ", allSwitchIps)
}

func CheckPreIsCollector(collectorip string) bool {
	if net.ParseIP(collectorip) == nil {
		return false
	}
	addr := g.Config().Http.Listen
	p := strings.Index(addr, ":")
	if p < 0 {
		return false
	}
	port := addr[p:]
	uri := fmt.Sprintf("http://%s%s/iscollector", collectorip, port)
	req := httplib.Get(uri).SetTimeout(1*time.Second, 1*time.Second)
	rstr, err := req.String()
	if err != nil {
		//第一次超时时间比较短可能出现丢包情况引起误切。增加一次探测
		req = httplib.Get(uri).SetTimeout(10*time.Second, 30*time.Second)
		rstr, err = req.String()
		if err != nil {
			return false
		}
	}
	return rstr == "true"

}

func CheckCollector() {
	if len(myExternalIps) == 0 {
		return
	}
	myip := myExternalIps[0]
	myindex := sort.SearchStrings(aliveList, myip)
	if !IsCollector {
		if myindex <= 1 {
			log.Println("I am collector!")
			IsCollector = true
			go StartLanSWcollect()
			return
		}
	}
	i := 0
	firstcollector := ""
	collectorNum := 0
	for ; i < myindex; i++ {
		if firstcollector != aliveList[i] && CheckPreIsCollector(aliveList[i]) {
			log.Println("The collector is !", aliveList[i])
			firstcollector = aliveList[i]
			collectorNum++
		}
		if collectorNum >= 1 {
			if IsCollector {
				log.Println("I am not collector now. ")
				IsCollector = false
				StopLanSWCollect()
			}
			return
		}
	}
	if !IsCollector {
		if i == myindex {
			log.Println("I am collector now!")
			IsCollector = true
			go StartLanSWcollect()
			return
		}
	} else {
		if isdebug {
			log.Println("I am still collector!")
		}
	}

}

func CronUpdateSwCollector() {
	for {
		UpdateSwList()
		if len(allSwitchIps) > 0 {
			CheckCollector()
		} else {
			if IsCollector {
				log.Println("I am not collector now. ")
				IsCollector = false
				StopLanSWCollect()
			}
		}
		time.Sleep(time.Second * 250)
	}

}

func InitLanIps() {
	myips, nets := ExternalIP()
	sort.Strings(myips)
	myExternalIps = myips
	//lanNetIps = make([]string, 0)
	IsCollector = false
	for _, inet := range nets {
		hostip, err := Hosts(inet)
		if err != nil {
			log.Println("Get net ips err:", err.Error())
			continue
		}
		//log.Println("get hostsip ", hostip)
		for _, h := range hostip {
			lanNetIps = append(lanNetIps, h)
		}

	}
	sort.Strings(lanNetIps)
	//log.Println("want to colletor ", lanNetIps)
	go CronUpdateSwCollector()
}
