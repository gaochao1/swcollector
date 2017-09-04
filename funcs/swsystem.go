package funcs

import (
	"log"

	"github.com/gaochao1/sw"
	"github.com/gaochao1/swcollector/g"
)

type SwSystem struct {
	Ip       string `json:"ip"`
	Hostname string `json:"hostname"`
	Model    string `json:"model"`
	Uptime   string `json:"uptime"`
	Cpu      int    `json:"cpu"`
	Mem      int    `json:"mem"`
	Ping     string `json:"ping"`
	Conn     int    `json:"Conn"`
}

func SwSystemInfo() (swList []SwSystem) {

	chs := make([]chan SwSystem, len(AliveIp))
	for i, ip := range AliveIp {
		chs[i] = make(chan SwSystem)
		go swSystemInfo(ip, chs[i])
	}

	for _, ch := range chs {
		swSystem := <-ch
		swList = append(swList, swSystem)
	}

	return swList
}

func swSystemInfo(ip string, ch chan SwSystem) {
	var swSystem SwSystem
	swSystem.Ip = ip

	//ping timeout.Millisecond
	timeout := 1000
	pingCount := 1

	ping, err := sw.PingStatSummary(ip, pingCount, timeout)
	if err != nil {
		log.Println(err)
		ch <- swSystem
		return
	} else {
		swSystem.Ping = ping["max"]

		uptime, err := sw.SysUpTime(ip, g.Config().Switch.Community, timeout)
		if err != nil {
			log.Println(err)
			ch <- swSystem
			return
		} else {
			swSystem.Uptime = uptime

			cpuUtili, err := sw.CpuUtilization(ip, g.Config().Switch.Community, timeout, 1)
			if err != nil {
				log.Println(err)
			} else {
				swSystem.Cpu = cpuUtili
			}

			memUtili, err := sw.MemUtilization(ip, g.Config().Switch.Community, timeout, 1)
			if err != nil {
				log.Println(err)
			} else {
				swSystem.Mem = memUtili
			}

			swModel, err := sw.SysModel(ip, g.Config().Switch.Community, timeout, 1)
			if err != nil {
				log.Println(err)
			} else {
				swSystem.Model = swModel
			}

			swName, err := sw.SysName(ip, g.Config().Switch.Community, timeout)
			if err != nil {
				log.Println(err)
			} else {
				swSystem.Hostname = swName
			}

		}

	}

	ch <- swSystem
	return
}
