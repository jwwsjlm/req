package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	collector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
)

// TestTraceProviderOTLP 验证官方环境变量、OTLP 导出与 service.name 资源属性。
func TestTraceProviderOTLP(t *testing.T) {
	requests := make(chan *collector.ExportTraceServiceRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/traces" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		data, err := io.ReadAll(r.Body)
		var request collector.ExportTraceServiceRequest
		if err != nil || proto.Unmarshal(data, &request) != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- &request
		w.Header().Set("Content-Type", "application/x-protobuf")
	}))
	defer server.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	tp, err := traceProvider()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, span := tp.Tracer("test").Start(ctx, "export")
	span.End()
	if err := tp.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case request := <-requests:
		for _, rs := range request.ResourceSpans {
			for _, attr := range rs.GetResource().GetAttributes() {
				if attr.Key == "service.name" && attr.Value.GetStringValue() == serviceName {
					return
				}
			}
		}
		t.Fatal("export lost service.name")
	default:
		t.Fatal("no OTLP request received")
	}
}
