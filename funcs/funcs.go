package funcs

import (
	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/open-falcon/common/model"
)

type FuncsAndInterval struct {
	Fs       []func() []*model.MetricValue
	Interval int
}

var Mappers []FuncsAndInterval

func BuildMappers() {
	interval := g.Config().Transfer.Interval
	Mappers = []FuncsAndInterval{
		FuncsAndInterval{
			Fs: []func() []*model.MetricValue{
				//SwIfMetrics,
				AgentMetrics,
				//CpuMetrics,
				//MemMetrics,
				PingMetrics,
				TrafficMetrics,
			},
			Interval: interval,
		},
	}

}
