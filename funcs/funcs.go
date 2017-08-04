package funcs

import (
	"github.com/gaochao1/swcollector/g"
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
				SwIfMetrics,
				CpuMetrics,
				MemMetrics,
				PingMetrics,
				CustMetrics,
			},
			Interval: interval,
		},
	}
}
