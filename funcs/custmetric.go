package funcs

import (
	"errors"
	"log"
	"strconv"

	"time"

	go_snmp "github.com/gaochao1/gosnmp"
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

func InArray(str string, array []string) bool {
	for _, s := range array {
		if str == s {
			return true
		}
	}
	return false
}

func AllCustmIp(ipRange []string) (allIp []string) {
	if len(ipRange) > 0 {
		for _, sip := range ipRange {
			aip := sw.ParseIp(sip)
			for _, ip := range aip {
				allIp = append(allIp, ip)
			}
		}
	}
	return allIp
}

func CustMetrics() (L []*model.MetricValue) {
	if !g.Config().CustomMetrics.Enabled {
		return
	}
	chs := make([]chan CustM, 0)
	for _, ip := range AliveIp {
		if ip != "" {
			for _, metric := range g.CustConfig().Metrics {
				CustmIps := AllCustmIp(metric.IpRange)
				if InArray(ip, CustmIps) {
					chss := make(chan CustM)
					go custMetrics(ip, metric, chss)
					chs = append(chs, chss)
				}
			}

		}
	}
	for _, ch := range chs {
		custm, ok := <-ch
		if !ok {
			continue
		}
		for _, custmmetric := range custm.custmMetrics {
			if custmmetric.metrictype == "GAUGE" {
				L = append(L, GaugeValueIp(time.Now().Unix(), custm.Ip, custmmetric.metric, custmmetric.value, custmmetric.tag))
			}
			if custmmetric.metrictype == "COUNTER" {
				L = append(L, CounterValueIp(time.Now().Unix(), custm.Ip, custmmetric.metric, custmmetric.value, custmmetric.tag))
			}

		}

	}

	return L
}

func custMetrics(ip string, metric *g.MetricConfig, ch chan CustM) {
	var custm CustM
	var custmmetric CustmMetric
	var custmmetrics []CustmMetric
	value, err := GetCustMetric(ip, g.Config().Switch.Community, metric.Oid, g.Config().Switch.SnmpTimeout, g.Config().Switch.SnmpRetry)
	if err != nil {
		log.Println(ip, metric.Oid, err)
		close(ch)
		return
	} else {
		custmmetric.metric = metric.Metric
		custmmetric.metrictype = metric.Type
		custmmetric.tag = metric.Tag
		custmmetric.value = value
		custmmetrics = append(custmmetrics, custmmetric)
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
	var snmpPDUs []go_snmp.SnmpPDU
	for i := 0; i < retry; i++ {
		snmpPDUs, err = sw.RunSnmp(ip, community, oid, method, timeout)
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
	case string:
		value_parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, err
		} else {
			return value_parsed, nil
		}
	default:
		err = errors.New("value cannot not Parse to digital")
		return 0, err
	}
}
