package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

func NewMeterProvider(serviceName string) (*metric.MeterProvider, error) {
	exp, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	r, err := NewResource(serviceName)
	if err != nil {
		return nil, err
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(exp),
		metric.WithResource(r),
	)

	otel.SetMeterProvider(mp)

	return mp, nil
}
