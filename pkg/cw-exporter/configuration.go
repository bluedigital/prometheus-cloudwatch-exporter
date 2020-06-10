package cwexporter

import (
	"reflect"
)

type metricSpec struct {
	name         string
	unit         string
	statistic    string
	promName     string
	timestamp    bool
	customLabels map[string]string
}

type metric struct {
	namespace  string
	specs      []*metricSpec
	period     int
	delay      int
	regions    []string
	dimensions []string
}

type configuration struct {
	metrics []*metric
	regions []string
}

func newConfiguration() *configuration {
	return &configuration{}
}

func (c *configuration) parse(source interface{}) error {

	config, ok := source.(map[string]interface{})
	if !ok {
		return ErrInvalidConfig
	}

	if r, ok := config["regions"]; ok {
		if v, ok := r.([]string); ok {
			c.regions = v
		}
	}

	if r, ok := config["metrics"]; ok {
		if v, ok := r.([]interface{}); ok {
			if err := c.parseMetrics(v); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *configuration) parseMetrics(source []interface{}) error {

	metrics := []*metric{}

	for _, mSource := range source {
		m := reflect.ValueOf(mSource)
		if m.Kind() != reflect.Map {
			return ErrInvalidConfig
		}

		metric := &metric{
			delay:  600,
			period: 60,
		}

		r := m.MapIndex(reflect.ValueOf("namespace"))

		if r.IsValid() {
			if v, ok := r.Interface().(string); ok {
				metric.namespace = v
			}
		}

		r = m.MapIndex(reflect.ValueOf("delay"))

		if r.IsValid() {
			if v, ok := r.Interface().(int); ok {
				metric.delay = v
			}
		}

		r = m.MapIndex(reflect.ValueOf("period"))

		if r.IsValid() {
			if v, ok := r.Interface().(int); ok {
				metric.period = v
			}
		}

		r = m.MapIndex(reflect.ValueOf("regions"))

		if r.IsValid() {
			if v, ok := r.Interface().([]interface{}); ok {
				for _, it := range v {
					metric.regions = append(metric.regions, it.(string))
				}
			}
		}

		r = m.MapIndex(reflect.ValueOf("dimensions"))

		if r.IsValid() {
			if v, ok := r.Interface().([]interface{}); ok {
				for _, it := range v {
					metric.dimensions = append(metric.dimensions, it.(string))
				}
			}
		}

		r = m.MapIndex(reflect.ValueOf("specs"))

		if r.IsValid() {
			if v, ok := r.Interface().([]interface{}); ok {
				for _, si := range v {
					sm := reflect.ValueOf(si)
					if sm.Kind() != reflect.Map {
						continue
					}
					s, err := c.parseMetricSpec(sm)
					if err != nil {
						return err
					}
					metric.specs = append(metric.specs, s)
				}
			}
		}

		metrics = append(metrics, metric)
	}

	c.metrics = metrics

	return nil
}

func (c *configuration) parseMetricSpec(source reflect.Value) (*metricSpec, error) {

	s := &metricSpec{
		timestamp: false,
	}

	r := source.MapIndex(reflect.ValueOf("name"))

	if r.IsValid() {
		if v, ok := r.Interface().(string); ok {
			s.name = v
		}
	}

	r = source.MapIndex(reflect.ValueOf("unit"))

	if r.IsValid() {
		if v, ok := r.Interface().(string); ok {
			s.unit = v
		}
	}

	r = source.MapIndex(reflect.ValueOf("statistic"))

	if r.IsValid() {
		if v, ok := r.Interface().(string); ok {
			s.statistic = v
		}
	}

	r = source.MapIndex(reflect.ValueOf("promName"))

	if r.IsValid() {
		if v, ok := r.Interface().(string); ok {
			s.promName = v
		}
	}

	r = source.MapIndex(reflect.ValueOf("timestamp"))

	if r.IsValid() {
		if v, ok := r.Interface().(bool); ok {
			s.timestamp = v
		}
	}

	r = source.MapIndex(reflect.ValueOf("customLabels"))

	if r.IsValid() {
		if v, ok := r.Interface().(map[interface{}]interface{}); ok {
			s.customLabels = make(map[string]string)
			for k, l := range v {
				s.customLabels[k.(string)] = l.(string)
			}
		}
	}

	return s, nil

}
