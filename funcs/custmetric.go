package funcs

import (
	"errors"
	"log"

	"time"

	"github.com/gaochao1/sw"
	"github.com/gaochao1/swcollector/g"
	"github.com/open-falcon/common/model"
)

type CustM struct {
	Ip           string
	custmMetrics []CustmMetric
}
type CustmMetric struct {
	metric     string
	tag        string
	value      float64
	metrictype string
}

func CustMetrics() (L []*model.MetricValue) {
	chs := make([]chan CustM, len(AliveIp))
	for i, ip := range AliveIp {
		if ip != "" {
			chs[i] = make(chan CustM)
			go custMetrics(ip, chs[i])
		}
	}
	for _, ch := range chs {
		custm := <-ch
		for _, custmmetric := range custm.custmMetrics {
			if custmmetric.metrictype == "GUAGE" {
				L = append(L, GaugeValueIp(time.Now().Unix(), custm.Ip, custmmetric.metric, custmmetric.value, custmmetric.tag))
			}
			if custmmetric.metrictype == "COUNTER" {
				L = append(L, CounterValueIp(time.Now().Unix(), custm.Ip, custmmetric.metric, custmmetric.value, custmmetric.tag))
			}

		}

	}

	return L
}

func custMetrics(ip string, ch chan CustM) {
	var custm CustM
	var custmmetric CustmMetric
	var custmmetrics []CustmMetric

	for _, metric := range g.CustConfig().Metrics {
		value, err := GetCustMetric(ip, g.Config().Switch.Community, metric.Oid, g.Config().Switch.SnmpTimeout, g.Config().Switch.SnmpRetry)
		if err != nil {
			log.Println(err)
		} else {
			custmmetric.metric = metric.Metric
			custmmetric.metrictype = metric.Type
			custmmetric.tag = metric.Tag
			custmmetric.value = value
			custmmetrics = append(custmmetrics, custmmetric)
		}
	}
	custm.Ip = ip
	custm.custmMetrics = custmmetrics
	ch <- custm
	return
}

func GetCustMetric(ip, community, oid string, timeout, retry int) (float64, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(ip+" Recovered in CustomMetric, Oid is ", oid, r)
		}
	}()
	method := "get"
	var value float64
	var err error
	for i := 0; i < retry; i++ {
		snmpPDUs, err := sw.RunSnmp(ip, community, oid, method, timeout)
		if len(snmpPDUs) > 0 && err == nil {
			value, err = interfaceTofloat64(snmpPDUs[0].Value)
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return value, err
}

func interfaceTofloat64(v interface{}) (float64, error) {
	var err error
	switch value := v.(type) {
	case int:
		return float64(value), nil
	case int8:
		return float64(value), nil
	case int16:
		return float64(value), nil
	case int32:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case uint:
		return float64(value), nil
	case uint8:
		return float64(value), nil
	case uint16:
		return float64(value), nil
	case uint32:
		return float64(value), nil
	case uint64:
		return float64(value), nil
	case float32:
		return float64(value), nil
	case float64:
		return value, nil
	default:
		err = errors.New("value is not digital")
		return 0, err
	}
}
