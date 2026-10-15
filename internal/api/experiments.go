package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Experiment struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type createExperimentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func ExperimentsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listExperiments(w, r, db)

		case http.MethodPost:
			createExperiment(w, r, db)

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
		}
	}
}

func createExperiment(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	var input createExperimentRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	if input.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name is required",
		})
		return
	}

	var experiment Experiment

	err := db.QueryRow(
		r.Context(),
		`
		INSERT INTO experiments (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, created_at
		`,
		input.Name,
		input.Description,
	).Scan(
		&experiment.ID,
		&experiment.Name,
		&experiment.Description,
		&experiment.CreatedAt,
	)

	if err != nil {
		log.Printf("create experiment: %v", err)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not create experiment",
		})
		return
	}

	writeJSON(w, http.StatusCreated, experiment)
}

func listExperiments(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	rows, err := db.Query(
		r.Context(),
		`
		SELECT id, name, description, created_at
		FROM experiments
		ORDER BY created_at DESC
		`,
	)

	if err != nil {
		log.Printf("list experiments: %v", err)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not list experiments",
		})
		return
	}
	defer rows.Close()

	experiments := make([]Experiment, 0)

	for rows.Next() {
		var experiment Experiment

		if err := rows.Scan(
			&experiment.ID,
			&experiment.Name,
			&experiment.Description,
			&experiment.CreatedAt,
		); err != nil {
			log.Printf("scan experiment: %v", err)

			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "could not read experiments",
			})
			return
		}

		experiments = append(experiments, experiment)
	}

	if err := rows.Err(); err != nil {
		log.Printf("read experiments: %v", err)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not read experiments",
		})
		return
	}

	writeJSON(w, http.StatusOK, experiments)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("write response: %v", err)
	}
}
