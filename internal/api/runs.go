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

type Run struct {
	ID           int64      `json:"id"`
	ExperimentID int64      `json:"experiment_id"`
	Name         string     `json:"name"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}

type createRunRequest struct {
	ExperimentID int64  `json:"experiment_id"`
	Name         string `json:"name"`
}

func RunsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listRuns(w, r, db)

		case http.MethodPost:
			createRun(w, r, db)

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
		}
	}
}

func createRun(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	var input createRunRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	input.Name = strings.TrimSpace(input.Name)

	if input.ExperimentID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "experiment_id is required",
		})
		return
	}

	if input.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name is required",
		})
		return
	}

	var run Run

	err := db.QueryRow(
		r.Context(),
		`
		INSERT INTO runs (experiment_id, name)
		VALUES ($1, $2)
		RETURNING id, experiment_id, name, status, started_at, finished_at
		`,
		input.ExperimentID,
		input.Name,
	).Scan(
		&run.ID,
		&run.ExperimentID,
		&run.Name,
		&run.Status,
		&run.StartedAt,
		&run.FinishedAt,
	)

	if err != nil {
		log.Printf("create run: %v", err)

		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "could not create run",
		})
		return
	}

	writeJSON(w, http.StatusCreated, run)
}

func listRuns(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	experimentID, err := strconv.ParseInt(
		r.URL.Query().Get("experiment_id"),
		10,
		64,
	)

	if err != nil || experimentID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "valid experiment_id is required",
		})
		return
	}

	rows, err := db.Query(
		r.Context(),
		`
		SELECT id, experiment_id, name, status, started_at, finished_at
		FROM runs
		WHERE experiment_id = $1
		ORDER BY started_at DESC
		`,
		experimentID,
	)

	if err != nil {
		log.Printf("list runs: %v", err)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not list runs",
		})
		return
	}
	defer rows.Close()

	runs := make([]Run, 0)

	for rows.Next() {
		var run Run

		if err := rows.Scan(
			&run.ID,
			&run.ExperimentID,
			&run.Name,
			&run.Status,
			&run.StartedAt,
			&run.FinishedAt,
		); err != nil {
			log.Printf("scan run: %v", err)

			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "could not read runs",
			})
			return
		}

		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		log.Printf("read runs: %v", err)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not read runs",
		})
		return
	}

	writeJSON(w, http.StatusOK, runs)
}
