package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// --- internal/store as sqlc writes it (db.go, models.go, requirements.sql.go), typed out.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Queries struct{ db DBTX }

func New(db DBTX) *Queries { return &Queries{db: db} }

type Requirement struct { // one field per column, in table order (the table is abridged)
	ID        uuid.UUID
	ProjectID uuid.UUID
	ParentID  *uuid.UUID // parent_id is nullable
	Code      string
	Title     string
	Status    string
}

// From this query in queries/requirements.sql:
//
//	-- name: CreateRequirement :one
//	INSERT INTO requirements (project_id, parent_id, code, title)
//	VALUES (@project_id, @parent_id, @code, @title) RETURNING *;
const createRequirement = `-- name: CreateRequirement :one
INSERT INTO requirements (project_id, parent_id, code, title)
VALUES ($1, $2, $3, $4) RETURNING id, project_id, parent_id, code, title, status
`

type CreateRequirementParams struct {
	ProjectID uuid.UUID
	ParentID  *uuid.UUID
	Code      string
	Title     string
}

func (q *Queries) CreateRequirement(ctx context.Context, arg CreateRequirementParams) (Requirement, error) {
	row := q.db.QueryRow(ctx, createRequirement, arg.ProjectID, arg.ParentID, arg.Code, arg.Title)
	var i Requirement
	err := row.Scan(&i.ID, &i.ProjectID, &i.ParentID, &i.Code, &i.Title, &i.Status)
	return i, err
}

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err == nil {
		_, err = conn.Exec(ctx, schema)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	q := New(conn) // a pool, a connection or a transaction: all satisfy DBTX
	project := uuid.MustParse("01937b6e-0000-7000-8000-00000000000a")
	auth, err := q.CreateRequirement(ctx, CreateRequirementParams{project, nil, "AUTH", "Kimlik doğrulama"})
	fmt.Printf("AUTH: parent=%v status=%s err=%v\n", auth.ParentID, auth.Status, err)
	_, err = q.CreateRequirement(ctx, CreateRequirementParams{project, nil, "AUTH", "Yine"})
	fmt.Println("duplicate:", err) // the driver's error, untranslated
}

const schema = `CREATE TABLE projects (id uuid PRIMARY KEY);
INSERT INTO projects VALUES ('01937b6e-0000-7000-8000-00000000000a');
CREATE TABLE requirements (id uuid PRIMARY KEY DEFAULT uuidv7(), project_id uuid NOT NULL REFERENCES projects,
  parent_id uuid, code text COLLATE "C" NOT NULL, title text NOT NULL, status text NOT NULL DEFAULT 'draft',
  CONSTRAINT requirements_code_key UNIQUE (project_id, code))`
