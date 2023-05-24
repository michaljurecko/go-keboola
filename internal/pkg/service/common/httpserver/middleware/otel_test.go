package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dimfeld/httptreemux/v5"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	otelTrace "go.opentelemetry.io/otel/trace"

	"github.com/keboola/keboola-as-code/internal/pkg/service/common/httpserver/middleware"
	"github.com/keboola/keboola-as-code/internal/pkg/telemetry"
)

const (
	responseContent = "some error"
)

func TestOpenTelemetryMiddleware(t *testing.T) {
	t.Parallel()

	// Setup tracing
	tel := telemetry.NewForTest(t)

	// Create muxer
	mux := httptreemux.NewContextMux()
	mux.UseHandler(middleware.OpenTelemetryExtractRoute())
	handler := middleware.Wrap(
		mux,
		middleware.RequestInfo(),
		middleware.OpenTelemetry(
			tel.TracerProvider(), tel.MeterProvider(),
			middleware.WithRedactedRouteParam("secret1"),
			middleware.WithRedactedQueryParam("secret2"),
			middleware.WithRedactedHeader("X-StorageAPI-Token"),
			middleware.WithFilter(func(req *http.Request) bool {
				return req.URL.Path != "/api/ignored"
			}),
		),
	)

	// Register endpoints
	grp := mux.NewGroup("/api")
	grp.GET("/ignored", func(w http.ResponseWriter, req *http.Request) {
		_, span := tel.Tracer().Start(req.Context(), "my-ignored-span")
		span.End(nil)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	grp.POST("/item/:id/:secret1", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(responseContent))
	})

	// Send request
	rec := httptest.NewRecorder()
	body := io.NopCloser(strings.NewReader("some body"))
	req := httptest.NewRequest("POST", "/api/item/123/my-secret-1?foo=bar&secret2=my-secret-2", body)
	req.Header.Set("User-Agent", "my-user-agent")
	req.Header.Set("X-StorageAPI-Token", "my-token")
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "some error", rec.Body.String())

	// Send ignored request
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/api/ignored", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())

	// Assert
	tel.AssertSpans(t, expectedSpans(tel), telemetry.WithAttributeMapper(func(attr attribute.KeyValue) attribute.KeyValue {
		if attr.Key == "http.request_id" && len(attr.Value.AsString()) > 0 {
			return attribute.String(string(attr.Key), "<dynamic>")
		}
		if attr.Key == "http.response.header.x-request-id" && len(attr.Value.AsString()) > 0 {
			return attribute.String(string(attr.Key), "<dynamic>")
		}
		return attr
	}))
	tel.AssertMetrics(t, expectedMetrics())
}

func expectedSpans(tel telemetry.ForTest) tracetest.SpanStubs {
	req1Context := otelTrace.NewSpanContext(otelTrace.SpanContextConfig{
		TraceID:    tel.TraceID(1),
		SpanID:     tel.SpanID(1),
		TraceFlags: otelTrace.FlagsSampled,
	})
	req2Context := otelTrace.NewSpanContext(otelTrace.SpanContextConfig{
		TraceID:    tel.TraceID(2),
		SpanID:     tel.SpanID(2),
		TraceFlags: otelTrace.FlagsSampled,
	})
	return tracetest.SpanStubs{
		{
			Name:        "http.server.request",
			SpanKind:    otelTrace.SpanKindServer,
			SpanContext: req1Context,
			Status: trace.Status{
				Code:        codes.Error,
				Description: "",
			},
			Attributes: []attribute.KeyValue{
				attribute.String("http.method", "POST"),
				attribute.String("http.scheme", "http"),
				attribute.String("http.flavor", "1.1"),
				attribute.String("net.host.name", "example.com"),
				attribute.String("net.sock.peer.addr", "192.0.2.1"),
				attribute.Int("net.sock.peer.port", 1234),
				attribute.String("http.user_agent", "my-user-agent"),
				attribute.String("http.request_id", "<dynamic>"),
				attribute.String("span.kind", "server"),
				attribute.String("span.type", "web"),
				attribute.String("http.query.foo", "bar"),
				attribute.String("http.query.secret2", "****"),
				attribute.String("http.header.x-storageapi-token", "****"),
				attribute.String("resource.name", "/api/item/:id/:secret1"),
				attribute.String("http.route", "/api/item/:id/:secret1"),
				attribute.String("http.route_param.id", "123"),
				attribute.String("http.route_param.secret1", "****"),
				attribute.String("http.response.header.x-request-id", "<dynamic>"),
				attribute.Int("http.wrote_bytes", 10),
				attribute.Int("http.status_code", http.StatusInternalServerError),
			},
		},
		{
			Name:           "http.server.request",
			SpanKind:       otelTrace.SpanKindInternal,
			SpanContext:    req2Context,
			ChildSpanCount: 1,
			Attributes: []attribute.KeyValue{
				attribute.Bool("manual.drop", true),
				attribute.String("resource.name", "/api/ignored"),
				attribute.String("http.route", "/api/ignored"),
			},
		},
		{
			Name:     "my-ignored-span",
			SpanKind: otelTrace.SpanKindInternal,
			Parent:   req2Context,
			SpanContext: otelTrace.NewSpanContext(otelTrace.SpanContextConfig{
				TraceID:    tel.TraceID(2),
				SpanID:     tel.SpanID(3),
				TraceFlags: otelTrace.FlagsSampled,
			}),
		},
	}
}

func expectedMetrics() []metricdata.Metrics {
	attrs := attribute.NewSet(
		attribute.String("http.method", "POST"),
		attribute.String("http.scheme", "http"),
		attribute.String("net.host.name", "example.com"),
		attribute.String("http.route", "/api/item/:id/:secret1"),
		attribute.Int("http.status_code", http.StatusInternalServerError),
	)
	return []metricdata.Metrics{
		{
			Name:        "keboola.go.http.server.request_content_length",
			Description: "",
			Data: metricdata.Sum[int64]{
				Temporality: 1,
				IsMonotonic: true, // counter
				DataPoints: []metricdata.DataPoint[int64]{
					{Value: 0, Attributes: attrs},
				},
			},
		},
		{
			Name:        "keboola.go.http.server.response_content_length",
			Description: "",
			Data: metricdata.Sum[int64]{
				Temporality: 1,
				IsMonotonic: true, // counter
				DataPoints: []metricdata.DataPoint[int64]{
					{Value: int64(len(responseContent)), Attributes: attrs},
				},
			},
		},
		{
			Name:        "keboola.go.http.server.duration",
			Description: "",
			Unit:        "",
			Data: metricdata.Histogram[float64]{
				Temporality: 1,
				DataPoints: []metricdata.HistogramDataPoint[float64]{
					{
						Count:      1,
						Bounds:     []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
						Attributes: attrs,
					},
				},
			},
		},
	}
}
