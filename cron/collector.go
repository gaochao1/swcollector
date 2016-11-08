package cron

import (
	"log"
	"math"
	"time"

	pfc "github.com/baishancloud/goperfcounter"
	"github.com/baishancloud/octopux-swcollector/funcs"
	"github.com/baishancloud/octopux-swcollector/funcs/lansw"
	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/open-falcon/common/model"
)

func Collect() {
	if !g.Config().Transfer.Enabled {
		return
	}

	if g.Config().Transfer.Addr == "" {
		return
	}
	go lansw.StartLanSWCollector()
	for _, v := range funcs.Mappers {
		go collect(int64(v.Interval), v.Fs)
	}
}

func collect(sec int64, fns []func() []*model.MetricValue) {
	for {
		go MetricToTransfer(sec, fns)
		time.Sleep(time.Duration(sec) * time.Second)
	}
}

func MetricToTransfer(sec int64, fns []func() []*model.MetricValue) {
	mvs := []*model.MetricValue{}
	ignoreMetrics := g.Config().IgnoreMetrics
	sec1 := int64(g.Config().Transfer.Interval)
	hostname, _ := g.Hostname()
	now := time.Now().Unix()
	for _, fn := range fns {
		items := fn()
		if items == nil {
			continue
		}

		if len(items) == 0 {
			continue
		}

		for _, mv := range items {
			if b, ok := ignoreMetrics[mv.Metric]; ok && b {
				continue
			} else {
				if mv.Step == 0 {
					mv.Step = sec1
					mv.Endpoint = hostname
					mv.Timestamp = now
				}
				mvs = append(mvs, mv)
			}
		}
	}

	startTime := time.Now()
	pfc.Meter("SWCLtfSend", int64(len(mvs)))
	//分批次传给transfer
	n := 5000
	lenMvs := len(mvs)

	div := lenMvs / n
	mod := math.Mod(float64(lenMvs), float64(n))

	mvsSend := []*model.MetricValue{}
	for i := 1; i <= div+1; i++ {

		if i < div+1 {
			mvsSend = mvs[n*(i-1) : n*i]
		} else {
			mvsSend = mvs[n*(i-1) : (n*(i-1))+int(mod)]
		}
		time.Sleep(100 * time.Millisecond)

		go g.SendToTransfer(mvsSend)
	}

	endTime := time.Now()
	log.Println("INFO : Send metrics to transfer running in the background. Process time :", endTime.Sub(startTime), "Send metrics :", len(mvs))
}
