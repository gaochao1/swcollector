package lansw

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	pfc "github.com/baishancloud/goperfcounter"
	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/dotwoo/smudge"
	"github.com/gaochao1/sw"
	"github.com/tevino/abool"
	tnet "github.com/toolkits/net"
	"github.com/toolkits/net/httplib"
)

var (
	lanNetIps      []string
	aliveList      []string
	healthList     []string
	healthlock     = new(sync.RWMutex)
	myExternalIps  []string
	myExternalNets []*net.IPNet
	localSwList    []string
	IsCollector    = abool.NewBool(false)
	Collector      string
	smudgePort     string
	smudgeIP       string
	CantSend       = false
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
	IP    string
	Alive bool
	IsSw  bool
}

func doPing(pingChan <-chan string, pongChan chan<- Pong) {
	healths := GetHeaths()
	for {
		ip, ok := <-pingChan
		if !ok {
			return
		}
		if contains(allSwitchIps, ip) {
			pongChan <- Pong{IP: ip, Alive: true, IsSw: true}
			continue
		}
		if contains(healths, ip) {
			pongChan <- Pong{IP: ip, Alive: true, IsSw: false}
			continue
		}
		alive, isSw := false, false
		_, err := exec.Command("ping", "-c1", "-t6", ip).Output()

		alive = (err == nil)
		if alive == false {
			pongChan <- Pong{IP: ip, Alive: alive, IsSw: isSw}
			continue
		}
		swVendor, _ := sw.SysVendor(ip, community, 1000)
		isSw = (len(swVendor) > 0 && swVendor != "Linux")
		pongChan <- Pong{IP: ip, Alive: alive, IsSw: isSw}
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
	alive := make([]string, 0)
	for _, p := range alives {
		if p.Alive && !p.IsSw {
			alive = append(alive, p.IP)
		} else if p.Alive && p.IsSw && !contains(localSwList, p.IP) {
			localSwList = append(localSwList, p.IP)
		}
	}
	sort.Strings(alive)
	aliveList = alive
	sort.Strings(localSwList)
	allSwitchIps = AllSwitchIP()
	log.Println("All switch ips : ", allSwitchIps)
}

func InitLanIps() {
	myExternalIps, myExternalNets = ExternalIP()
	sort.Strings(myExternalIps)
	for _, inet := range myExternalNets {
		hostip, err := Hosts(inet)
		if err != nil {
			log.Println("Get net ips err:", err.Error())
			continue
		}

		for _, h := range hostip {
			lanNetIps = append(lanNetIps, h)
		}

	}
	sort.Strings(lanNetIps)
	UpdateSwList()
	StartSmudge()
}

func NetStatusUpdate() {
	for {
		time.Sleep(10 * time.Minute)
		if IsCollector.IsSet() {
			UpdateSwList()
			allswip := strings.Join(allSwitchIps, ",")
			smudge.BroadcastString("switchlist," + allswip)
		} else {
			log.Println("the collector is ", Collector)
		}
	}
}

func TrySWCollect() {
	if len(myExternalIps) < 1 || CantSend {
		smudge.Stop()
		IsCollector.UnSet()
		return
	}
	//TODO:确认发送成功状态
	if IsCollector.IsSet() {
		return
	}
	IsCollector.Set()
	Collector = smudgeIP
	time.Sleep(10 * time.Second)
	if IsCollector.IsSet() {
		StartSWCollect()
	}
}

func StartSWCollect() {
	if len(myExternalIps) < 1 {
		return
	}
	IsCollector.Set()
	log.Println("I am collector now,", Collector)
	StartSWcollectTask()
}

func StopSWCollect(ip string) {
	Collector = ip
	IsCollector.UnSet()
	log.Println("i am not collector , collector is:", ip)
	//StopSWCollectTask()
}

func MonitSWcollector() {
	for {
		time.Sleep(2 * time.Minute)
		healths := GetHeaths()
		sort.Strings(healths)
		log.Printf("Im collector status :%t ,myip:%s, healths:%v\n", IsCollector.IsSet(), smudgeIP, healths)
		if !IsCollector.IsSet() {
			if len(healths) > 0 {
				Collector = healths[0]
			}
			if contains(myExternalIps, Collector) {
				StartSWCollect()
			}
			if len(healths) > 10 {
				myindex := sort.SearchStrings(healths, smudgeIP)
				if myindex >= 10 {
					log.Println("Too many smudge,im goto stop.")
					smudge.BroadcastString("stop")
					smudge.Stop()
				}
			}
			continue
		}
		UpdateSwList()
		if CantSend == true && pfc.GetMeterRate1("SWCLSendS") > 1/60 && pfc.GetMeterRate15("SWCLSwSendFails") <= 1/60 {
			CantSend = false
			go smudge.Begin()
		}
		// checkout send health
		if pfc.GetMeterRate1("SWCLSwSendFails") > pfc.GetMeterRate1("SWCLSwSend")/2 || pfc.GetMeterRate5("SWCLSwSendFails") > pfc.GetMeterRate5("SWCLSwSend")/2 {
			log.Printf("I am send sw faile. status is F1m:%6.4f > C1m: %6.4f /2 ,F5m:%6.4f > C5m: %6.4f /2 , F15m:%6.4f ,C15m: %6.4f \n",
				pfc.GetMeterRate1("SWCLSwSendFails"),
				pfc.GetMeterRate1("SWCLSwSend"),
				pfc.GetMeterRate5("SWCLSwSendFails"),
				pfc.GetMeterRate5("SWCLSwSend"),
				pfc.GetMeterRate15("SWCLSwSendFails"),
				pfc.GetMeterRate15("SWCLSwSend"),
			)
			if len(healths) > 1 {
				log.Println("My smudge goto exit.")
				smudge.BroadcastString("break")
				CantSend = true
				time.Sleep(5 * time.Second)
				smudge.Stop()
				Collector = ""
				IsCollector.UnSet()
				log.Print("i am not collector, i am break.")
			}
			//StopSWCollectTask()
			break
		}

		if len(healths) >= 1 && smudgeIP != healths[0] {
			if IsCollector.IsSet() {
				StopSWCollect(healths[0])
			}
		}

	}
}

type MyStatusListener struct {
	smudge.StatusListener
}

func UpdateHealthList() {
	healthl := make([]string, 0)
	hnodes := smudge.HealthyNodes()
	for _, node := range hnodes {
		healthl = append(healthl, node.IP().String())
	}
	sort.Strings(healthl)
	healthlock.Lock()
	defer healthlock.Unlock()
	healthList = healthl
}
func GetHeaths() []string {
	healthlock.RLock()
	defer healthlock.RUnlock()
	return healthList
}

func checkCollectorAlive(ip string) bool {
	if net.ParseIP(ip) == nil {
		return false
	}
	addr := g.Config().Http.Listen
	p := strings.Index(addr, ":")
	if p < 0 {
		return false
	}
	port := addr[p:]
	uri := fmt.Sprintf("http://%s%s/iscollector", ip, port)
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

func (m MyStatusListener) OnChange(node *smudge.Node, status smudge.NodeStatus) {
	//log.Printf("Node %s is now status %s\n", node.Address(), status)
	UpdateHealthList()
	healths := GetHeaths()
	if !CantSend {
		if !contains(healths, smudgeIP) {
			healths = append(healths, smudgeIP)
		}
	}
	sort.Strings(healths)
	if status == smudge.StatusDead {
		log.Printf("OnChange %s::%s,myip:%s,healths:%v\n", node.IP().String(), status.String(), smudgeIP, healths)
		if node.IP().String() == Collector {
			if checkCollectorAlive(node.IP().String()) {
				smudge.UpdateNodeStatus(node, smudge.StatusAlive)
				return
			}
		}
	}
	if len(healths) <= 1 || smudgeIP == healths[0] {
		TrySWCollect()
	} else {
		if IsCollector.IsSet() {
			StopSWCollect(healths[0])
		}
	}
}

type MyBroadcastListener struct {
	smudge.BroadcastListener
}

func (m MyBroadcastListener) OnBroadcast(b *smudge.Broadcast) {
	// log.Printf("Received broadcast from %s: %s\n",
	// 	b.Origin().Address(), string(b.Bytes()))
	UpdateInfo(string(b.Bytes()), b.Origin().IP())
}

func UpdateInfo(info string, ip net.IP) {
	items := strings.Split(info, ",")
	if len(items) < 1 {
		return
	}
	switch items[0] {
	case "break":
		log.Println("Receive ip break ", ip.String())
		pfc.Report("SendBreak,ip"+ip.String(), ip.String()+"cantSendSwFlow")
	case "switchlist":
		UpdateSwitchInfo(items[1:])

	}
}

func UpdateSwitchInfo(newswips []string) {
	log.Println("UpdateSwitchInfo:", newswips)
	nofindSwip := make([]string, 0)
	for _, ip := range allSwitchIps {
		if !contains(newswips, ip) {
			nofindSwip = append(nofindSwip, ip)
		}
	}
	if len(nofindSwip) > 0 {
		allswip := strings.Join(nofindSwip, ",")
		smudge.BroadcastString("switchlist," + allswip)
	}
	for _, ip := range newswips {
		if !contains(allSwitchIps, ip) {
			allSwitchIps = append(allSwitchIps, ip)
			log.Println("Find news switch ip from smadge.", ip)
		}
	}
	sort.Strings(allSwitchIps)
}

func StartSmudge() {
	conf := g.Config()
	if conf.Switch == nil || !conf.Switch.Enabled {
		return
	}

	if len(myExternalIps) < 1 {
		log.Println("not have external ip, cant collector snmp")
		return
	}

	heartbeatMillis := conf.Switch.Heartbeat
	if heartbeatMillis < 10 {
		heartbeatMillis = 500
	}
	listenPort := conf.Switch.SmudgePort

	smudgePort = fmt.Sprintf(":%d", listenPort)
	// Set configuration options
	smudge.SetListenPort(listenPort)
	smudge.SetHeartbeatMillis(heartbeatMillis)

	sort.Strings(myExternalIps)
	smudeglip := net.ParseIP(myExternalIps[0])
	if smudeglip == nil {
		smudeglip, _ = smudge.GetLocalIP()
	}
	smudge.SetListenIP(smudeglip)
	smudgeIP = smudeglip.String()

	// Add the status listener
	smudge.AddStatusListener(MyStatusListener{})

	// Add the broadcast listener
	smudge.AddBroadcastListener(MyBroadcastListener{})

	// Add a new remote node. Currently, to join an existing cluster you must
	// add at least one of its healthy member nodes.
	for _, peer := range aliveList {
		node, err := smudge.CreateNodeByAddress(peer + smudgePort)
		if !contains(myExternalIps, peer) {
			smudge.UpdateNodeStatus(node, smudge.StatusDead)
		}
		if err == nil {
			smudge.AddNode(node)
		}
	}
	smudge.SetLogThreshold(smudge.LogError)
	go smudge.Begin()

	time.Sleep(30 * time.Second)
	go MonitSWcollector()
	UpdateHealthList()
	IsCollector.UnSet()
	TrySWCollect()
}
