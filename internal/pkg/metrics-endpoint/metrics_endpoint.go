package metricsendpoint

import (
	"context"
	"net/http"

	cwexporter "github.com/bluedigital/prometheus-cloudwatch-exporter/pkg/cw-exporter"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"

	"github.com/rs/zerolog/log"
)

type MetricsEndpoint struct {
	server      *http.Server
	exporter    *cwexporter.Exporter
	promHandler http.Handler
}

func NewMetricsEndpoint(e *cwexporter.Exporter) *MetricsEndpoint {
	return &MetricsEndpoint{
		exporter:    e,
		promHandler: promhttp.Handler(),
	}
}

func (m *MetricsEndpoint) Start() {
	r := mux.NewRouter()
	r.HandleFunc("/metrics", m.MetricsHandler)
	r.HandleFunc("/reload", m.ReloadHandler)

	m.server = &http.Server{
		Addr:              viper.GetString("metrics_address"),
		Handler:           r,
		ReadTimeout:       viper.GetDuration("metrics_read_timeout"),
		ReadHeaderTimeout: viper.GetDuration("metrics_read_header_timeout"),
		IdleTimeout:       viper.GetDuration("metrics_idle_timeout"),
	}

	go func() {
		err := m.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Send()
		}
	}()
}

func (m *MetricsEndpoint) Stop() {

	localCtx, localCancel := context.WithTimeout(context.Background(), viper.GetDuration("metrics_shutdown_timeout"))
	defer localCancel()
	err := m.server.Shutdown(localCtx)
	if err == nil {
		return
	}
	log.Error().Err(err).Send()
	err = m.server.Close()
	if err == nil {
		return
	}
	log.Error().Err(err).Send()

}

func (m *MetricsEndpoint) MetricsHandler(req http.ResponseWriter, res *http.Request) {
	m.exporter.UpdateCache()
	m.promHandler.ServeHTTP(req, res)
	log.Info().Msg("metrics endpoint scraped")
}

func (m *MetricsEndpoint) ReloadHandler(req http.ResponseWriter, res *http.Request) {
	err := m.exporter.Load()
	if err != nil {
		log.Error().Err(err).Msg("reload error")
	}
	m.promHandler.ServeHTTP(req, res)
	log.Info().Msg("reload enpoint called")
}
