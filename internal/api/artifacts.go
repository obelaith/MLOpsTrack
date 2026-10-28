package api

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Artifact struct {
	ID         int64     `json:"id"`
	RunID      int64     `json:"run_id"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func validArtifactName(name string) bool {
	if name == "" || name == "." || name == ".." || len(name) > 100 {
		return false
	}

	for _, c := range name {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '.' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func ArtifactsHandler(db *pgxpool.Pool, root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed,
				map[string]string{"error": "method not allowed"})
			return
		}

		runID, err := strconv.ParseInt(r.URL.Query().Get("run_id"), 10, 64)
		if err != nil || runID <= 0 {
			writeJSON(w, http.StatusBadRequest,
				map[string]string{"error": "valid run_id is required"})
			return
		}

		name := r.URL.Query().Get("name")

		if r.Method == http.MethodPost {
			if !validArtifactName(name) {
				writeJSON(w, http.StatusBadRequest,
					map[string]string{"error": "invalid artifact name"})
				return
			}
			uploadArtifact(w, r, db, root, runID, name)
			return
		}

		if name == "" {
			listArtifacts(w, r, db, runID)
			return
		}

		if !validArtifactName(name) {
			writeJSON(w, http.StatusBadRequest,
				map[string]string{"error": "invalid artifact name"})
			return
		}

		downloadArtifact(w, r, db, root, runID, name)
	}
}

func uploadArtifact(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, root string, runID int64, name string) {
	var exists bool
	err := db.QueryRow(r.Context(),
		"SELECT EXISTS (SELECT 1 FROM runs WHERE id = $1)", runID,
	).Scan(&exists)

	if err != nil {
		log.Printf("check run: %v", err)
		writeJSON(w, http.StatusInternalServerError,
			map[string]string{"error": "could not check run"})
		return
	}
	if !exists {
		writeJSON(w, http.StatusNotFound,
			map[string]string{"error": "run not found"})
		return
	}

	dir := filepath.Join(root, strconv.FormatInt(runID, 10))
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Printf("create artifact directory: %v", err)
		writeJSON(w, http.StatusInternalServerError,
			map[string]string{"error": "could not create directory"})
		return
	}

	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

	if errors.Is(err, os.ErrExist) {
		writeJSON(w, http.StatusConflict,
			map[string]string{"error": "artifact already exists"})
		return
	}
	if err != nil {
		log.Printf("create artifact: %v", err)
		writeJSON(w, http.StatusInternalServerError,
			map[string]string{"error": "could not create artifact"})
		return
	}

	size, copyErr := io.Copy(file, http.MaxBytesReader(w, r.Body, 100<<20))
	closeErr := file.Close()

	if copyErr != nil || closeErr != nil || size == 0 {
		os.Remove(path)

		var tooLarge *http.MaxBytesError
		if errors.As(copyErr, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge,
				map[string]string{"error": "artifact exceeds 100 MiB"})
		} else if copyErr == nil && closeErr == nil {
			writeJSON(w, http.StatusBadRequest,
				map[string]string{"error": "artifact is empty"})
		} else {
			log.Printf("write artifact: %v, %v", copyErr, closeErr)
			writeJSON(w, http.StatusInternalServerError,
				map[string]string{"error": "could not save artifact"})
		}
		return
	}

	var artifact Artifact
	err = db.QueryRow(r.Context(), `
		INSERT INTO artifacts (run_id, name, size)
		VALUES ($1, $2, $3)
		RETURNING id, run_id, name, size, uploaded_at
	`, runID, name, size).Scan(
		&artifact.ID, &artifact.RunID, &artifact.Name,
		&artifact.Size, &artifact.UploadedAt,
	)

	if err != nil {
		os.Remove(path)
		log.Printf("save artifact record: %v", err)
		writeJSON(w, http.StatusInternalServerError,
			map[string]string{"error": "could not save artifact record"})
		return
	}

	writeJSON(w, http.StatusCreated, artifact)
}

func listArtifacts(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, runID int64) {
	rows, err := db.Query(r.Context(), `
		SELECT id, run_id, name, size, uploaded_at
		FROM artifacts
		WHERE run_id = $1
		ORDER BY id
	`, runID)

	if err != nil {
		log.Printf("list artifacts: %v", err)
		writeJSON(w, http.StatusInternalServerError,
			map[string]string{"error": "could not list artifacts"})
		return
	}
	defer rows.Close()

	artifacts := make([]Artifact, 0)
	for rows.Next() {
		var artifact Artifact
		err := rows.Scan(
			&artifact.ID, &artifact.RunID, &artifact.Name,
			&artifact.Size, &artifact.UploadedAt,
		)
		if err != nil {
			log.Printf("scan artifact: %v", err)
			writeJSON(w, http.StatusInternalServerError,
				map[string]string{"error": "could not read artifacts"})
			return
		}
		artifacts = append(artifacts, artifact)
	}

	if err := rows.Err(); err != nil {
		log.Printf("read artifacts: %v", err)
		writeJSON(w, http.StatusInternalServerError,
			map[string]string{"error": "could not read artifacts"})
		return
	}

	writeJSON(w, http.StatusOK, artifacts)
}

func downloadArtifact(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, root string, runID int64, name string) {
	var id int64
	err := db.QueryRow(r.Context(),
		"SELECT id FROM artifacts WHERE run_id = $1 AND name = $2",
		runID, name,
	).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound,
			map[string]string{"error": "artifact not found"})
		return
	}
	if err != nil {
		log.Printf("find artifact: %v", err)
		writeJSON(w, http.StatusInternalServerError,
			map[string]string{"error": "could not find artifact"})
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	http.ServeFile(w, r,
		filepath.Join(root, strconv.FormatInt(runID, 10), name))
}
