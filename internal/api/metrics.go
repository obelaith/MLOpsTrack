package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Metric struct {
	ID         int64     `json:"id"`
	RunID      int64     `json:"run_id"`
	Name       string    `json:"name"`
	Value      float64   `json:"value"`
	Step       int64     `json:"step"`
	RecordedAt time.Time `json:"recorded_at"`
}

type createMetricRequest struct {
	RunID int64   `json:"run_id"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Step  int64   `json:"step"`
}

func MetricsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listMetrics(w, r, db)

		case http.MethodPost:
			createMetric(w, r, db)

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
		}
	}
}

func createMetric(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	var input createMetricRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	input.Name = strings.TrimSpace(input.Name)

	if input.RunID <= 0 || input.Name == "" || input.Step < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "valid run_id, name, and step are required",
		})
		return
	}

	var metric Metric

	err := db.QueryRow(
		r.Context(),
		`
		INSERT INTO metrics (run_id, name, value, step)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (run_id, name, step)
		DO UPDATE SET
			value = EXCLUDED.value,
			recorded_at = NOW()
		RETURNING id, run_id, name, value, step, recorded_at
		`,
		input.RunID,
		input.Name,
		input.Value,
		input.Step,
	).Scan(
		&metric.ID,
		&metric.RunID,
		&metric.Name,
		&metric.Value,
		&metric.Step,
		&metric.RecordedAt,
	)

	if err != nil {
		log.Printf("create metric: %v", err)

		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "could not save metric",
		})
		return
	}

	writeJSON(w, http.StatusCreated, metric)
}

func listMetrics(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	runID, err := strconv.ParseInt(r.URL.Query().Get("run_id"), 10, 64)

	if err != nil || runID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "valid run_id is required",
		})
		return
	}

	rows, err := db.Query(
		r.Context(),
		`
		SELECT id, run_id, name, value, step, recorded_at
		FROM metrics
		WHERE run_id = $1
		ORDER BY name, step
		`,
		runID,
	)

	if err != nil {
		log.Printf("list metrics: %v", err)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not list metrics",
		})
		return
	}
	defer rows.Close()

	metrics := make([]Metric, 0)

	for rows.Next() {
		var metric Metric

		if err := rows.Scan(
			&metric.ID,
			&metric.RunID,
			&metric.Name,
			&metric.Value,
			&metric.Step,
			&metric.RecordedAt,
		); err != nil {
			log.Printf("scan metric: %v", err)

			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "could not read metrics",
			})
			return
		}

		metrics = append(metrics, metric)
	}

	writeJSON(w, http.StatusOK, metrics)
}
