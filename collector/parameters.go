package collector

import (
	"strings"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metric_mapping = map[string]prometheus.Gauge{}
)

type ParameterMetrics struct {
}

func (p *ParameterMetrics) Export(ch chan<- prometheus.Metric) {
	for _, metric := range metric_mapping {
		metric.Collect(ch)
	}
}

func (p *ParameterMetrics) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range metric_mapping {
		metric.Describe(ch)
	}
}

func GetParameters(session *mgo.Session, parameters string) *ParameterMetrics {
	split_parameters := strings.Split(parameters, ",")
	for _, parameter := range split_parameters {
		if _, ok := metric_mapping[parameter]; !ok {
			metric_mapping[parameter] = prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: "parameters",
				Name:      parameter,
				Help:      "A setParamter option in mongod",
			})
		}
		metric := metric_mapping[parameter]
		result := make(map[string]interface{})
		err := session.DB("admin").Run(bson.D{{"getParameter", 1}, {parameter, 1}}, result)
		if err != nil {
			glog.Error("Failed to get parameter value for %v: %v", parameter, err)
			continue
		}
		if val, ok := result[parameter]; ok {
			switch valTyped := val.(type) {
			case int:
				metric.Set(float64(valTyped))
			case float64:
				metric.Set(valTyped)
			case bool:
				var bit int8
				if valTyped {
					bit = 1
				}
				metric.Set(float64(bit))
			}
		} else {
			glog.Error("Unexpected response from getParameter command: %v", result)
		}
	}
	return &ParameterMetrics{}
}
