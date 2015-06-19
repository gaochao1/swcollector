package funcs

import (
	"github.com/gaochao1/swcollector/g"
	"github.com/open-falcon/common/model"
	"strings"
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

func GaugeValue(metric string, val interface{}, tags ...string) *model.MetricValue {
	return NewMetricValue(metric, val, "GAUGE", tags...)
}

func CounterValue(metric string, val interface{}, tags ...string) *model.MetricValue {
	return NewMetricValue(metric, val, "COUNTER", tags...)
}

func NewMetricValueIp(TS int64, ip, metric string, val interface{}, dataType string, tags ...string) *model.MetricValue {
	sec := int64(g.Config().Transfer.Interval)

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
		mv.Tags = strings.Join(tags, ",")
	}

	return &mv
}

func GaugeValueIp(TS int64, ip, metric string, val interface{}, tags ...string) *model.MetricValue {
	return NewMetricValueIp(TS, ip, metric, val, "GAUGE", tags...)
}

func CounterValueIp(TS int64, ip, metric string, val interface{}, tags ...string) *model.MetricValue {
	return NewMetricValueIp(TS, ip, metric, val, "COUNTER", tags...)
}
