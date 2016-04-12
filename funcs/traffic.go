package funcs

import (
	"log"
	"net"

	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/open-falcon/common/model"
	"github.com/toolkits/nux"
	//"fmt"
	"os/exec"
	"strconv"
	"strings"
)

var lanlist []net.IPNet

func getfaces() map[string]string {
	list, err := net.Interfaces()
	if err != nil {
		log.Println("ERROR: get interfaces!", err)
		return nil
		//panic(err)
	}
	if len(lanlist) == 0 {
		if g.Config().Collector == nil || g.Config().Collector.LanIpnet == nil {
			return nil
		}
		//println("init lanlist")
		lanstrs := g.Config().Collector.LanIpnet
		parseIPNets(&lanstrs, &lanlist)

	}
	tiflist := make(map[string]string)
	//Ifacelies =  make([]string,0)
	for _, iface := range list {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 || (iface.Flags&net.FlagPointToPoint) != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if len(addrs) == 0 {
			//println("no ip")
			continue
		}
		if err != nil {
			continue
		}
		islan := false
		for _, addr := range addrs {

			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ip.To4() == nil {
				continue
			}
			//println(len(lanlist))
			for _, in := range lanlist {
				//println(ip.String(),in.String())
				if in.Contains(ip) {
					islan = true
					break
				}
			}
			if islan == false {
				tiflist[ip.String()] = iface.Name
			}

		}
	}
	return tiflist
}

func parseIPNets(lanstrs *[]string, nw *[]net.IPNet) {

	rows := len(*lanstrs)
	if rows == 0 {
		return
	}
	nw1 := make([]net.IPNet, 0, rows)
	i := 0
	for _, s := range *lanstrs {
		_, ipNet, err := net.ParseCIDR(s)
		if err == nil {
			i++
			nw1 = append(nw1, *ipNet)
		}
	}
	*nw = (nw1[:i])
	return
}

func resetipt() {
	cmdrt := exec.Command("/usr/local/bin/ipt_server.sh")
	_, err := cmdrt.Output()
	if err != nil {
		log.Println("reset iptable error", err)
	}
}

func gettraffic(iface string) (wi string, li string, wo string, lo string, err error) {
	iptablesbinary, lookErr := exec.LookPath("iptables")
	if lookErr != nil {
		//panic(lookErr)
		return "", "", "", "", lookErr
	}
	//println(ip, iface)
	cmdai := exec.Command(iptablesbinary, "-L", "traffic_in_"+iface, "1", "-vnx")
	cmdli := exec.Command(iptablesbinary, "-L", "traffic_lan_in_"+iface, "1", "-vnx")
	cmdao := exec.Command(iptablesbinary, "-L", "traffic_out_"+iface, "1", "-vnx")
	cmdlo := exec.Command(iptablesbinary, "-L", "traffic_lan_out_"+iface, "1", "-vnx")

	aiOut, err := cmdai.Output()
	if err != nil {
		//resetipt()
		return "", "", "", "", err
	}
	line := string(aiOut)
	fis := strings.Fields(line)
	ai := fis[1]
	//fmt.Printf("Fields are: %s\n",aiv )

	liOut, err := cmdli.Output()
	if err != nil {
		//resetipt()
		return "", "", "", "", err
	}
	line = string(liOut)
	fis = strings.Fields(line)
	li = fis[1]
	//println(liv)

	aoOut, err := cmdao.Output()
	if err != nil {
		//resetipt()
		return "", "", "", "", err
	}
	line = string(aoOut)
	fis = strings.Fields(line)
	ao := fis[1]
	//println(aov)
	loOut, err := cmdlo.Output()
	if err != nil {
		//resetipt()
		return "", "", "", "", err
	}
	line = string(loOut)
	fis = strings.Fields(line)
	lo = fis[1]
	aiv, err := strconv.ParseUint(ai, 10, 64)
	if err != nil {
		return "", "", "", "", err
	}
	liv, err := strconv.ParseUint(li, 10, 64)
	if err != nil {
		return "", "", "", "", err
	}
	aov, err := strconv.ParseUint(ao, 10, 64)
	if err != nil {
		return "", "", "", "", err
	}
	lov, err := strconv.ParseUint(lo, 10, 64)
	if err != nil {
		return "", "", "", "", err
	}
	wiv := aiv - liv
	wov := aov - lov

	wi = strconv.FormatUint(wiv, 10)
	wo = strconv.FormatUint(wov, 10)

	return wi, li, wo, lo, nil

}

func TrafficMetrics() (L []*model.MetricValue) {
	return CoreTrafficMetrics()
}

func CoreTrafficMetrics() (L []*model.MetricValue) {
	tifs := getfaces()
	if tifs == nil || len(tifs) == 0 {
		log.Println("get faces error")
		return
	}
	ifacePrefix := make([]string, 0, 0)
	for _, v := range tifs {
		isexist := false
		for _, ifa := range ifacePrefix {
			if ifa == v {
				isexist = true
				break
			}
		}
		if isexist == false {
			ifacePrefix = append(ifacePrefix, v)
		}
	}
	netIfs, err := nux.NetIfs(ifacePrefix)
	if err != nil {
		log.Println(err)
		return []*model.MetricValue{}
	}

	for _, netIf := range netIfs {
		ifacetag := "iface=" + netIf.Iface
		L = append(L, CounterValue("netcard.if.in.bytes", netIf.InBytes, ifacetag))
		L = append(L, CounterValue("netcard.if.out.bytes", netIf.OutBytes, ifacetag))
		L = append(L, CounterValue("netcard.if.total.bytes", netIf.TotalBytes, ifacetag))
	}

	for ip, iface := range tifs {
		wi, li, wo, lo, err := gettraffic(iface)
		iptag := "ip=" + ip
		ifacetag := "iface=" + iface
		if err != nil {
			log.Println("get taffic falure. ", err)
			L = append(L, GaugeValue("traffic.collect.status", 1, ifacetag, iptag))
			continue
		}
		L = append(L, GaugeValue("traffic.collect.status", 0, ifacetag, iptag))
		L = append(L, CounterValue("traffic.wan.in", wi, ifacetag, iptag))
		L = append(L, CounterValue("traffic.lan.in", li, ifacetag, iptag))
		L = append(L, CounterValue("traffic.wan.out", wo, ifacetag, iptag))
		L = append(L, CounterValue("traffic.lan.out", lo, ifacetag, iptag))
	}
	return L
}
