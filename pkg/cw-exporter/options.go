package cwexporter

import (
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type ExporterOption func(*Exporter)

func WithLogger(l *zerolog.Logger) ExporterOption {
	return func(e *Exporter) {
		e.log = l
	}
}

func WithViper(v *viper.Viper) ExporterOption {
	return func(e *Exporter) {
		e.viper = v
	}
}
