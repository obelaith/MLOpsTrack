package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Param struct {
	ID    int64  `json:"id"`
	RunID int64  `json:"run_id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type createParamRequest struct {
	RunID int64  `json:"run_id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

func ParamsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listParams(w, r, db)

		case http.MethodPost:
			createParam(w, r, db)

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
		}
	}
}

func createParam(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	var input createParamRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	input.Key = strings.TrimSpace(input.Key)
	input.Value = strings.TrimSpace(input.Value)

	if input.RunID <= 0 || input.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "run_id and key are required",
		})
		return
	}

	var param Param

	err := db.QueryRow(
		r.Context(),
		`
		INSERT INTO run_params (run_id, key, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (run_id, key)
		DO UPDATE SET value = EXCLUDED.value
		RETURNING id, run_id, key, value
		`,
		input.RunID,
		input.Key,
		input.Value,
	).Scan(
		&param.ID,
		&param.RunID,
		&param.Key,
		&param.Value,
	)

	if err != nil {
		log.Printf("create param: %v", err)

		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "could not save parameter",
		})
		return
	}

	writeJSON(w, http.StatusCreated, param)
}

func listParams(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
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
		SELECT id, run_id, key, value
		FROM run_params
		WHERE run_id = $1
		ORDER BY key
		`,
		runID,
	)

	if err != nil {
		log.Printf("list params: %v", err)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not list parameters",
		})
		return
	}
	defer rows.Close()

	params := make([]Param, 0)

	for rows.Next() {
		var param Param

		if err := rows.Scan(
			&param.ID,
			&param.RunID,
			&param.Key,
			&param.Value,
		); err != nil {
			log.Printf("scan param: %v", err)

			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "could not read parameters",
			})
			return
		}

		params = append(params, param)
	}

	writeJSON(w, http.StatusOK, params)
}
