package cwexporter

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type Exporter struct {
	log          *zerolog.Logger
	session      *session.Session
	viper        *viper.Viper
	walkDuration time.Duration
	walkIncr     int
	config       *configuration
	regionalCW   map[string]*cloudwatch.CloudWatch
	collector    *PrometheusCollector
}

func NewExporter(options ...ExporterOption) *Exporter {
	e := &Exporter{
		regionalCW:   make(map[string]*cloudwatch.CloudWatch),
		walkDuration: viper.GetDuration("aws_walk_scrape"),
	}

	for _, opt := range options {
		opt(e)
	}

	return e
}

func (e *Exporter) Load() error {
	creds := credentials.NewStaticCredentialsFromCreds(credentials.Value{
		ProviderName:    "CWExporter",
		AccessKeyID:     viper.GetString("aws_access_key_id"),
		SecretAccessKey: viper.GetString("aws_secret_access_key"),
		SessionToken:    viper.GetString("aws_session_token"),
	})
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(viper.GetString("aws_region")),
		MaxRetries:  aws.Int(viper.GetInt("aws_max_retries")),
		Credentials: creds,
	})
	if err != nil {
		if e.log != nil {
			e.log.Fatal().Err(err).Send()
		}
		return err
	}
	e.session = sess

	err = e.viper.ReadInConfig()
	if err != nil {
		return err
	}
	c := e.viper.Get("exporter")

	if c == nil {
		if e.log != nil {
			e.log.Fatal().Err(ErrMissingConfig).Send()
		}
		return ErrMissingConfig
	}

	e.config = &configuration{}
	err = e.config.parse(c)

	if err != nil {
		if e.log != nil {
			e.log.Error().Err(err).Send()
		}
		return err
	}

	if e.collector != nil {
		e.collector.reset()
	} else {
		e.collector = NewPrometheusCollector()
	}

	if e.log != nil {
		e.log.Info().Msg("prometheus cloudwatch exporter loaded")
	}

	return nil
}

func (e *Exporter) UpdateCache() {

	queries := map[string][]*cloudwatch.MetricDataQuery{}

	e.collector.reset()

	for mi, m := range e.config.metrics {
		for si, s := range m.specs {

			regions := []string{viper.GetString("aws_region")}
			if len(m.regions) > 0 {
				regions = m.regions
			} else if len(e.config.regions) > 0 {
				regions = m.regions
			}

			for ri, r := range regions {
				if _, ok := queries[r]; !ok {
					queries[r] = make([]*cloudwatch.MetricDataQuery, 0)
				}

				listMetricsIn := &cloudwatch.ListMetricsInput{
					MetricName: aws.String(s.name),
					Namespace:  aws.String(m.namespace),
				}

				cw := e.getRegionalCloudWatch(r)

				err := cw.ListMetricsPages(listMetricsIn, func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
					var buf bytes.Buffer
					buf.WriteString("cw_")
					buf.WriteString(strconv.Itoa(mi))
					buf.WriteRune('_')
					buf.WriteString(strconv.Itoa(si))
					buf.WriteRune('_')
					buf.WriteString(strconv.Itoa(ri))
					return e.handleMetricsPageOutput(page, lastPage, buf.String(), m, s, r)
				})

				if err != nil {
					if e.log != nil {
						e.log.Error().Err(err).Send()
					} else {
						fmt.Printf(err.Error())
					}
				}

			}
		}
	}

	err := prometheus.Register(e.collector)

	if err != nil {
		if e.log != nil {
			e.log.Error().Err(err).Send()
		} else {
			fmt.Println(err)
		}
	}

	e.regionalCW = make(map[string]*cloudwatch.CloudWatch)
}

func (e *Exporter) handleMetricsPageOutput(page *cloudwatch.ListMetricsOutput, lastPage bool, id string, mt *metric, s *metricSpec, r string) bool {

	dm := make(map[string]bool)

	for _, d := range mt.dimensions {
		dm[d] = true
	}

	idMap := make(map[string]int)
	qs := []*cloudwatch.MetricDataQuery{}

	for mi, m := range page.Metrics {
		mid := fmt.Sprintf("%vM%03d", id, mi)
		q := &cloudwatch.MetricDataQuery{
			Id:    aws.String(mid),
			Label: aws.String(s.promName),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(mt.namespace),
					MetricName: aws.String(s.name),
				},
				Period: aws.Int64(int64(mt.period)),
				Stat:   aws.String(s.statistic),
				Unit:   aws.String(s.unit),
			},
			ReturnData: aws.Bool(true),
		}

		for _, d := range m.Dimensions {
			if _, ok := dm[aws.StringValue(d.Name)]; !ok {
				continue
			}
			q.MetricStat.Metric.Dimensions = append(
				q.MetricStat.Metric.Dimensions,
				d,
			)
		}

		idMap[mid] = len(qs)
		qs = append(qs, q)

		// AWS hard limit
		if len(qs) == 500 {

			if err := e.getMetricData(qs, mt, s, r, idMap); err != nil {
				return false
			}

			idMap = make(map[string]int)
			qs = []*cloudwatch.MetricDataQuery{}
		}

	}

	if len(qs) > 0 {
		if err := e.getMetricData(qs, mt, s, r, idMap); err != nil {
			return false
		}
	}

	return !lastPage
}

func (e *Exporter) getMetricData(qs []*cloudwatch.MetricDataQuery, mt *metric, s *metricSpec, r string, idMap map[string]int) error {
	cw := e.getRegionalCloudWatch(r)

	d := time.Duration(mt.delay) * time.Second
	period := time.Duration(mt.period) * time.Second
	now := time.Now().UTC()
	startTime := now.Add(-d)
	endTime := startTime.Add(period)

	if e.walkDuration > 0 {
		e.walkIncr++
		incr := time.Duration(e.walkIncr * int(mt.period))
		endTime = endTime.Add(incr)
		if endTime.After(now.Add(-e.walkDuration)) {
			e.walkDuration = 0
			endTime = endTime.Add(-period)
		} else {
			startTime = startTime.Add(incr)
		}
	}

	getMetricDataIn := &cloudwatch.GetMetricDataInput{
		MetricDataQueries: qs,
		ScanBy:            aws.String("TimestampDescending"),
		StartTime:         &startTime,
		EndTime:           &endTime,
	}

	err := cw.GetMetricDataPages(getMetricDataIn, func(page *cloudwatch.GetMetricDataOutput, lastPage bool) bool {
		return e.handleMetricDataOutput(page, lastPage, qs, mt, s, idMap)
	})

	if err != nil {
		if e.log != nil {
			e.log.Error().Err(err).Send()
		} else {
			fmt.Printf(err.Error())
		}
		return err
	}

	return nil
}

func (e *Exporter) handleMetricDataOutput(page *cloudwatch.GetMetricDataOutput, lastPage bool, qs []*cloudwatch.MetricDataQuery, mt *metric, s *metricSpec, idMap map[string]int) bool {

	for _, r := range page.MetricDataResults {
		if len(r.Values) == 0 {
			continue
		}

		promName := aws.StringValue(r.Label)
		id := aws.StringValue(r.Id)
		q := qs[idMap[id]]

		custLabelsLen := 0

		if s.customLabels != nil {
			custLabelsLen = len(s.customLabels)
		}

		labels := make(prometheus.Labels, len(q.MetricStat.Metric.Dimensions))
		for _, d := range q.MetricStat.Metric.Dimensions {
			labels[aws.StringValue(d.Name)] = aws.StringValue(d.Value)
		}
		if custLabelsLen > 0 {
			for k, v := range s.customLabels {
				labels[k] = v
			}
		}
		g := prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        promName,
			ConstLabels: labels,
		})

		g.Set(aws.Float64Value(r.Values[0]))

		if !s.timestamp {
			e.collector.addMetric(g)
			continue
		}

		gts := prometheus.NewMetricWithTimestamp(aws.TimeValue(r.Timestamps[0]), g)

		e.collector.addMetric(gts)

	}

	return !lastPage
}

func (e *Exporter) getRegionalCloudWatch(region string) *cloudwatch.CloudWatch {
	cw, ok := e.regionalCW[region]
	if ok {
		return cw
	}
	cw = cloudwatch.New(
		e.session,
		&aws.Config{
			Region: aws.String(region),
		},
	)
	e.regionalCW[region] = cw
	return cw
}
