package funcs

import (
	"strings"

	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/open-falcon/common/model"
)

func NewMetricValue(metric string, val interface{}, dataType string, tags ...string) *model.MetricValue {
	mv := model.MetricValue{
		Metric: metric,
		Value:  val,
		Type:   dataType,
	}

	size := len(tags)

	if size > 0 {
		mv.Tags = strings.Join(tags, ",")
	}

	return &mv
}

func NewMetricValueSliceTS(metric string, ts []int64, val interface{}, dataType string, tags ...string) []*model.MetricValue {
	tag := ""
	size := len(tags)

	if size > 0 {
		tag = strings.Join(tags, ",")
	}
	mvs := []*model.MetricValue{}
	for _, t := range ts {
		mv := model.MetricValue{
			Metric:    metric,
			Value:     val,
			Type:      dataType,
			Timestamp: t,
			Tags:      tag,
		}
		mvs = append(mvs, &mv)
	}

	return mvs
}

func GaugeValue(metric string, val interface{}, tags ...string) *model.MetricValue {
	return NewMetricValue(metric, val, "GAUGE", tags...)
}

func GaugeValueSliceTS(metric string, ts []int64, val interface{}, tags ...string) []*model.MetricValue {
	return NewMetricValueSliceTS(metric, ts, val, "GAUGE", tags...)
}

func CounterValue(metric string, val interface{}, tags ...string) *model.MetricValue {
	return NewMetricValue(metric, val, "COUNTER", tags...)
}

func NewMetricValueIp(TS int64, ip, metric string, val interface{}, dataType string, tags ...string) *model.MetricValue {
	sec := int64(g.Config().Transfer.Interval)
	hostname, _ := g.Hostname()
	collectstr := "collecter=" + hostname
	mv := model.MetricValue{
		Metric:    metric,
		Value:     val,
		Type:      dataType,
		Endpoint:  ip,
		Step:      sec,
		Timestamp: TS,
	}

	size := len(tags)

	if size > 0 {
		tags = append(tags, collectstr)
		mv.Tags = strings.Join(tags, ",")
	} else {
		mv.Tags = collectstr
	}

	return &mv
}

func NewMetricValueIpSlicTs(ts []int64, ip, metric string, val interface{}, dataType string, tags ...string) []*model.MetricValue {
	sec := int64(g.Config().Transfer.Interval)
	hostname, _ := g.Hostname()
	collectstr := "collecter=" + hostname
	tag := ""
	size := len(tags)

	if size > 0 {
		tags = append(tags, collectstr)
		tag = strings.Join(tags, ",")
	} else {
		tag = collectstr
	}

	mvs := []*model.MetricValue{}
	for _, t := range ts {
		mv := model.MetricValue{
			Metric:    metric,
			Value:     val,
			Type:      dataType,
			Endpoint:  ip,
			Timestamp: t,
			Step:      sec,
			Tags:      tag,
		}
		mvs = append(mvs, &mv)
	}

	return mvs

}

func GaugeValueIp(TS int64, ip, metric string, val interface{}, tags ...string) *model.MetricValue {
	return NewMetricValueIp(TS, ip, metric, val, "GAUGE", tags...)
}

func GaugeValueIpSlicTs(ts []int64, ip, metric string, val interface{}, tags ...string) []*model.MetricValue {
	return NewMetricValueIpSlicTs(ts, ip, metric, val, "GAUGE", tags...)
}

func CounterValueIp(TS int64, ip, metric string, val interface{}, tags ...string) *model.MetricValue {
	return NewMetricValueIp(TS, ip, metric, val, "COUNTER", tags...)
}
