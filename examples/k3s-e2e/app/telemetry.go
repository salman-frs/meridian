package e2eapp

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type telemetry struct {
	tracer     trace.Tracer
	meter      metric.Meter
	shutdown   func(context.Context) error
	requests   metric.Int64Counter
	checkouts  metric.Int64Counter
	heartbeats metric.Int64Counter
}

func newTelemetry(ctx context.Context, cfg Config) (*telemetry, error) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logForced(cfg, "error", "otel_error", map[string]any{"error": err.Error()})
	}))

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			attribute.String("service.namespace", "meridian-e2e"),
			attribute.String("meridian.run_id", cfg.RunID),
			attribute.String("meridian.scenario", cfg.Scenario),
		),
	)
	if err != nil {
		return nil, err
	}

	shutdowns := make([]func(context.Context) error, 0, 2)

	if !cfg.DisableTraces {
		traceExporter, err := otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlptracegrpc.WithTimeout(cfg.OTLPTimeout),
			otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
				Enabled:         true,
				InitialInterval: 250 * time.Millisecond,
				MaxInterval:     time.Second,
				MaxElapsedTime:  5 * time.Second,
			}),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
		traceProvider := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(traceExporter),
		)
		otel.SetTracerProvider(traceProvider)
		shutdowns = append(shutdowns, traceProvider.Shutdown)
	}

	if !cfg.DisableMetrics {
		metricExporter, err := otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlpmetricgrpc.WithTimeout(cfg.OTLPTimeout),
			otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
				Enabled:         true,
				InitialInterval: 250 * time.Millisecond,
				MaxInterval:     time.Second,
				MaxElapsedTime:  5 * time.Second,
			}),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(
				sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(5*time.Second)),
			),
		)
		otel.SetMeterProvider(meterProvider)
		shutdowns = append(shutdowns, meterProvider.Shutdown)
	}

	tracer := otel.Tracer("meridian-k3s-e2e")
	meter := otel.Meter("meridian-k3s-e2e")

	requests, err := meter.Int64Counter("meridian_http_requests_total")
	if err != nil {
		return nil, err
	}
	checkouts, err := meter.Int64Counter("meridian_checkout_total")
	if err != nil {
		return nil, err
	}
	heartbeats, err := meter.Int64Counter("meridian_inventory_heartbeat_total")
	if err != nil {
		return nil, err
	}

	return &telemetry{
		tracer:     tracer,
		meter:      meter,
		requests:   requests,
		checkouts:  checkouts,
		heartbeats: heartbeats,
		shutdown: func(ctx context.Context) error {
			for i := len(shutdowns) - 1; i >= 0; i-- {
				if err := shutdowns[i](ctx); err != nil {
					return err
				}
			}
			return nil
		},
	}, nil
}

func (t *telemetry) client() *otelhttp.Transport {
	return otelhttp.NewTransport(nil)
}

func commonAttrs(cfg Config, extras ...attribute.KeyValue) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("run_id", cfg.RunID),
		attribute.String("scenario", cfg.Scenario),
		attribute.String("service", cfg.ServiceName),
	}
	attrs = append(attrs, extras...)
	return attrs
}

func commonMetricOption(cfg Config, extras ...attribute.KeyValue) metric.AddOption {
	return metric.WithAttributes(commonAttrs(cfg, extras...)...)
}
