package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvalidRequests(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
		body    string
		want    int
	}{
		{
			name:    "empty experiment name",
			handler: ExperimentsHandler(nil),
			method:  http.MethodPost,
			path:    "/experiments",
			body:    `{"name":"   "}`,
			want:    http.StatusBadRequest,
		},
		{
			name:    "unknown experiment field",
			handler: ExperimentsHandler(nil),
			method:  http.MethodPost,
			path:    "/experiments",
			body:    `{"name":"test","unknown":true}`,
			want:    http.StatusBadRequest,
		},
		{
			name:    "unsupported method",
			handler: ExperimentsHandler(nil),
			method:  http.MethodDelete,
			path:    "/experiments",
			want:    http.StatusMethodNotAllowed,
		},
		{
			name:    "invalid experiment ID",
			handler: RunsHandler(nil),
			method:  http.MethodPost,
			path:    "/runs",
			body:    `{"experiment_id":0,"name":"test"}`,
			want:    http.StatusBadRequest,
		},
		{
			name:    "empty run name",
			handler: RunsHandler(nil),
			method:  http.MethodPost,
			path:    "/runs",
			body:    `{"experiment_id":1,"name":" "}`,
			want:    http.StatusBadRequest,
		},
		{
			name:    "invalid run status",
			handler: RunsHandler(nil),
			method:  http.MethodPatch,
			path:    "/runs",
			body:    `{"id":1,"status":"invalid"}`,
			want:    http.StatusBadRequest,
		},
		{
			name:    "missing parameter key",
			handler: ParamsHandler(nil),
			method:  http.MethodPost,
			path:    "/params",
			body:    `{"run_id":1,"key":"","value":"0.001"}`,
			want:    http.StatusBadRequest,
		},
		{
			name:    "invalid metrics query",
			handler: MetricsHandler(nil),
			method:  http.MethodGet,
			path:    "/metrics?run_id=abc",
			want:    http.StatusBadRequest,
		},
		{
			name:    "negative metric step",
			handler: MetricsHandler(nil),
			method:  http.MethodPost,
			path:    "/metrics",
			body:    `{"run_id":1,"name":"loss","value":0.5,"step":-1}`,
			want:    http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(
				test.method,
				test.path,
				strings.NewReader(test.body),
			)

			response := httptest.NewRecorder()

			test.handler(response, request)

			if response.Code != test.want {
				t.Fatalf(
					"expected status %d, got %d: %s",
					test.want,
					response.Code,
					response.Body.String(),
				)
			}

			var result map[string]string

			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}

			if result["error"] == "" {
				t.Fatal("expected an error message")
			}
		})
	}
}
