package cwexporter

import "errors"

var (
	ErrInvalidConfig = errors.New("invalid configuration for cw exporter")
	ErrMissingConfig = errors.New("missing configuration for cw exporter")
)
