package telemetry

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerprom "github.com/uber/jaeger-lib/metrics/prometheus"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// ErrInvalidConfiguration is an error to notify client to provide valid trace report agent or config server
var (
	ErrBlankTraceConfiguration = errors.New("no trace report agent, config server, or collector endpoint specified")
)

// installJaeger registers Jaeger as the Opentelemetry implementation.
func installJaeger(serviceName string, cfg *jaegercfg.Configuration, options ...jaegercfg.Option) (io.Closer, error) {
	mProvider, err := NewMeterProvider(serviceName)
	exitOnError(err, "error setting up OTel for metrics")

	metricsFactory := jaegerprom.New()

	// put the metricsFactory earlier so provided options can override it
	opts := append([]jaegercfg.Option{jaegercfg.Metrics(metricsFactory)}, options...)

	closer, err := cfg.InitGlobalTracer(serviceName, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize jaeger tracer")
	}
	return closer, nil
}

// NewFromEnv is a convenience function to allow tracing configuration
// via environment variables
//
// Tracing will be enabled if one (or more) of the following environment variables is used to configure trace reporting:
// - JAEGER_AGENT_HOST
// - JAEGER_SAMPLER_MANAGER_HOST_PORT
func NewFromEnv(serviceName string, options ...jaegercfg.Option) (io.Closer, error) {
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		return nil, errors.Wrap(err, "could not load jaeger tracer configuration")
	}

	if cfg.Sampler.SamplingServerURL == "" && cfg.Reporter.LocalAgentHostPort == "" && cfg.Reporter.CollectorEndpoint == "" {
		return nil, ErrBlankTraceConfiguration
	}

	return installJaeger(serviceName, cfg, options...)
}

// ExtractTraceID extracts the trace id, if any from the context.
func ExtractTraceID(ctx context.Context) (string, bool) {
	sp := trace.SpanFromContext(ctx)
	traceId := sp.SpanContext().TraceID().String()
	if traceId == "" {
		return "", false
	}
	return traceId, true
}

// ExtractSampledTraceID works like ExtractTraceID but the returned bool is only
// true if the returned trace id is sampled.
func ExtractSampledTraceID(ctx context.Context) (string, bool) {
	sp := trace.SpanFromContext(ctx)
	traceId := sp.SpanContext().TraceID().String()
	if traceId == "" {
		return "", false
	}
	return traceId, sp.SpanContext().IsSampled()
}

func NewTracerProvider(serviceName string) (*sdktrace.TracerProvider, error) {
	exp, err := newExporter()
	if err != nil {
		return nil, err
	}

	r, err := NewResource(serviceName)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

func newExporter() (*otlptrace.Exporter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return otlptracegrpc.New(ctx)
}

func GetTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer("otlp-gateway")
}
