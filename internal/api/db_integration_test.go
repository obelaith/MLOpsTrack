package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/obelaith/MLOpsTrack/internal/db"
)

func TestDatabaseIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if config.ConnConfig.Database != "mlopstrack_test" {
		t.Fatal("integration tests require the mlopstrack_test database")
	}

	database, err := db.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)

	mux := http.NewServeMux()
	mux.HandleFunc("/experiments", ExperimentsHandler(database))
	mux.HandleFunc("/runs", RunsHandler(database))
	mux.HandleFunc("/params", ParamsHandler(database))
	mux.HandleFunc("/metrics", MetricsHandler(database))
	mux.HandleFunc("/artifacts", ArtifactsHandler(database, t.TempDir()))

	var experiment Experiment
	checkRequest(t, mux, "POST", "/experiments",
		`{"name":"integration-test"}`, 201, &experiment)
	if experiment.ID <= 0 {
		t.Fatal("experiment was not created")
	}

	t.Cleanup(func() {
		_, err := database.Exec(context.Background(),
			"DELETE FROM experiments WHERE id = $1", experiment.ID)
		if err != nil {
			t.Errorf("clean up experiment: %v", err)
		}
	})

	var run Run
	checkRequest(t, mux, "POST", "/runs",
		fmt.Sprintf(`{"experiment_id":%d,"name":"training"}`, experiment.ID),
		201, &run)
	if run.ID <= 0 || run.Status != "running" {
		t.Fatalf("unexpected new run: %+v", run)
	}

	var runs []Run
	checkRequest(t, mux, "GET", fmt.Sprintf("/runs?experiment_id=%d", experiment.ID),
		"", 200, &runs)
	if len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("unexpected runs: %+v", runs)
	}

	var firstParam, updatedParam Param
	checkRequest(t, mux, "POST", "/params",
		fmt.Sprintf(`{"run_id":%d,"key":"lr","value":"0.01"}`, run.ID),
		201, &firstParam)
	checkRequest(t, mux, "POST", "/params",
		fmt.Sprintf(`{"run_id":%d,"key":"lr","value":"0.1"}`, run.ID),
		201, &updatedParam)
	if updatedParam.ID != firstParam.ID || updatedParam.Value != "0.1" {
		t.Fatalf("parameter upsert failed: %+v", updatedParam)
	}

	var params []Param
	checkRequest(t, mux, "GET", fmt.Sprintf("/params?run_id=%d", run.ID),
		"", 200, &params)
	if len(params) != 1 || params[0].Value != "0.1" {
		t.Fatalf("unexpected parameters: %+v", params)
	}

	var firstMetric, updatedMetric Metric
	checkRequest(t, mux, "POST", "/metrics",
		fmt.Sprintf(`{"run_id":%d,"name":"loss","value":0.5,"step":1}`, run.ID),
		201, &firstMetric)
	checkRequest(t, mux, "POST", "/metrics",
		fmt.Sprintf(`{"run_id":%d,"name":"loss","value":0.25,"step":1}`, run.ID),
		201, &updatedMetric)
	if updatedMetric.ID != firstMetric.ID || updatedMetric.Value != 0.25 {
		t.Fatalf("metric upsert failed: %+v", updatedMetric)
	}

	var metrics []Metric
	checkRequest(t, mux, "GET", fmt.Sprintf("/metrics?run_id=%d", run.ID),
		"", 200, &metrics)
	if len(metrics) != 1 || metrics[0].Value != 0.25 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}

	artifactPath := fmt.Sprintf(
		"/artifacts?run_id=%d&name=model.pt", run.ID,
	)

	var artifact Artifact
	checkRequest(t, mux, "POST", artifactPath,
		"test model weights", 201, &artifact)

	if artifact.RunID != run.ID || artifact.Name != "model.pt" ||
		artifact.Size != int64(len("test model weights")) {
		t.Fatalf("unexpected artifact: %+v", artifact)
	}

	var artifacts []Artifact
	checkRequest(t, mux, "GET",
		fmt.Sprintf("/artifacts?run_id=%d", run.ID),
		"", 200, &artifacts)

	if len(artifacts) != 1 || artifacts[0].ID != artifact.ID {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}

	request := httptest.NewRequest("GET", artifactPath, nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != 200 || response.Body.String() != "test model weights" {
		t.Fatalf("artifact download failed: %d %s",
			response.Code, response.Body.String())
	}

	var finished Run
	checkRequest(t, mux, "PATCH", "/runs",
		fmt.Sprintf(`{"id":%d,"status":"completed"}`, run.ID),
		200, &finished)
	if finished.Status != "completed" || finished.FinishedAt == nil {
		t.Fatalf("run was not completed: %+v", finished)
	}
}

func checkRequest(t *testing.T, handler http.Handler, method, path, body string, status int, result any) {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != status {
		t.Fatalf("%s %s: expected %d, got %d: %s",
			method, path, status, response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), result); err != nil {
		t.Fatalf("%s %s: decode response: %v", method, path, err)
	}
}
