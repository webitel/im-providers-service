package postgresx

import (
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

type TracingOption func(*tracingConfig)

type tracingConfig struct {
	enabled          bool
	tracerProvider   *trace.TracerProvider
	includeQueryText bool
	sanitizeQuery    bool
	attrs            []attribute.KeyValue
}

func defaultTracingConfig() *tracingConfig {
	return &tracingConfig{
		tracerProvider:   trace.NewTracerProvider(),
		includeQueryText: true,
		sanitizeQuery:    false,
	}
}

func WithTracing(opts ...TracingOption) PoolOption {
	return func(cfg *pgxpool.Config) {
		tc := defaultTracingConfig()
		tc.enabled = true
		for _, o := range opts {
			o(tc)
		}
		cfg.ConnConfig.Tracer = buildTracer(tc)
	}
}

func WithTracerProvider(tp *trace.TracerProvider) TracingOption {
	return func(tc *tracingConfig) {
		tc.tracerProvider = tp
	}
}

func WithTraceQueryText(enabled bool) TracingOption {
	return func(tc *tracingConfig) {
		tc.includeQueryText = enabled
	}
}

func WithTraceSanitizeQuery(enabled bool) TracingOption {
	return func(tc *tracingConfig) {
		tc.sanitizeQuery = enabled
	}
}

func WithTraceAttributes(attrs ...attribute.KeyValue) TracingOption {
	return func(tc *tracingConfig) {
		tc.attrs = append(tc.attrs, attrs...)
	}
}

func buildTracer(tc *tracingConfig) pgx.QueryTracer {
	opts := []otelpgx.Option{
		otelpgx.WithTracerProvider(tc.tracerProvider),
	}
	if tc.includeQueryText {
		opts = append(opts, otelpgx.WithIncludeQueryParameters())
	}
	if len(tc.attrs) > 0 {
		opts = append(opts, otelpgx.WithTracerAttributes(tc.attrs...))
	}
	return otelpgx.NewTracer(opts...)
}
