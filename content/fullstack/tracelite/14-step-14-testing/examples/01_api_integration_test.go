package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type service struct{ db *pgxpool.Pool } // step 4's service, reduced to its pool

// handleCreate is step 4's handler, reduced to a rule only the database knows.
func (s service) handleCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code     string  `json:"code"`
		ParentID *string `json:"parentId"`
	}
	if err := json.UnmarshalRead(r.Body, &in, json.RejectUnknownMembers(true)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err := s.db.Exec(r.Context(), "INSERT INTO requirements (project_id, code, parent_id) VALUES ($1, $2, $3)",
		r.PathValue("projectId"), in.Code, in.ParentID)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.ConstraintName == "requirements_parent_fkey" {
		w.WriteHeader(http.StatusUnprocessableEntity) // step 4 writes a problem naming #/parentId
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}

var db *pgxpool.Pool
var api = http.NewServeMux() // the router

// TestMain runs once per test binary: one pool, the migrations (goose.Up in TraceLite), the routes.
func TestMain(m *testing.M) {
	var err error
	if db, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL")); err == nil {
		_, err = db.Exec(context.Background(), `CREATE TABLE requirements ( -- migration 001, abridged
			id uuid PRIMARY KEY DEFAULT uuidv7(), project_id uuid NOT NULL, parent_id uuid, code text NOT NULL, UNIQUE (project_id, id),
			CONSTRAINT requirements_parent_fkey FOREIGN KEY (project_id, parent_id) REFERENCES requirements (project_id, id))`)
	}
	if err != nil {
		panic(err)
	}
	api.HandleFunc("POST /api/projects/{projectId}/requirements", service{db}.handleCreate)
	os.Exit(m.Run())
}

// newProject commits a requirement, AUTH, in a project of the test's own.
func newProject(t *testing.T) (project, auth string) {
	row := db.QueryRow(t.Context(), "INSERT INTO requirements (project_id, code) VALUES (uuidv7(), 'AUTH') RETURNING project_id, id")
	if err := row.Scan(&project, &auth); err != nil {
		t.Fatal(err)
	}
	return project, auth
}

func create(project, parent string) (status int) {
	rec, body := httptest.NewRecorder(), strings.NewReader(`{"code": "AUTH-RESET", "parentId": "`+parent+`"}`)
	api.ServeHTTP(rec, httptest.NewRequest("POST", "/api/projects/"+project+"/requirements", body))
	return rec.Code
}

func TestCreateUnderParent(t *testing.T) {
	t.Parallel() // safe: every row this test reads or writes is in its own projects
	project, auth := newProject(t)
	_, elsewhere := newProject(t)
	if same, other := create(project, auth), create(project, elsewhere); same != 201 || other != 422 {
		t.Errorf("parent in the same project: %d, in another: %d; want 201 and 422", same, other)
	}
}
