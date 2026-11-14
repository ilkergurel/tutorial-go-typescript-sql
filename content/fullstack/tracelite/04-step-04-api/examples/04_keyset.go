package main

import (
	"cmp"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
)

type Requirement struct { // abridged to its sort key
	Code string `json:"code"`
}
type RequirementPage struct {
	Items      []Requirement `json:"items"`
	NextCursor *string       `json:"nextCursor"` // null on the last page
}

// listAfter stands in for the store: ... WHERE project_id = $1 AND code > $2 ORDER BY code LIMIT $3
func listAfter(after string, limit int) []Requirement {
	var rows []Requirement
	for _, code := range []string{"AUTH", "AUTH-LOGIN", "AUTH-RESET", "EXPORT", "EXPORT-CSV"} {
		if code > after && len(rows) < limit {
			rows = append(rows, Requirement{code})
		}
	}
	return rows
}

// GET /api/projects/{projectId}/requirements?limit=&cursor=
func listRequirements(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(cmp.Or(r.URL.Query().Get("limit"), "50")) // cmp.Or: the first non-empty
	if err != nil || limit < 1 || limit > 200 {
		writeProblem(w, http.StatusBadRequest, "limit must be an integer from 1 to 200")
		return
	}
	var after [1]string // the last code seen; "" without a cursor, and "" sorts first
	if s := r.URL.Query().Get("cursor"); s != "" {
		key, err := base64.RawURLEncoding.DecodeString(s)
		if err != nil || json.Unmarshal(key, &after) != nil {
			writeProblem(w, http.StatusBadRequest, "cursor is not one this server issued")
			return
		}
	}
	items := listAfter(after[0], limit+1)
	page := RequirementPage{Items: items}
	if len(items) > limit { // the extra row only says that a next page exists
		page.Items = items[:limit]
		key, _ := json.Marshal([1]string{items[limit-1].Code})
		next := base64.RawURLEncoding.EncodeToString(key)
		page.NextCursor = &next
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalWrite(w, page)
}

// writeProblem writes an about:blank problem; in TraceLite, writeError does (example 3).
func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.MarshalWrite(w, struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}{"about:blank", http.StatusText(status), status, detail})
}

func main() {
	for _, query := range []string{
		"limit=2", "limit=2&cursor=WyJBVVRILUxPR0lOIl0", "limit=2&cursor=WyJFWFBPUlQiXQ", "limit=2&cursor=WyJaIl0",
		"limit=2&cursor=bm90IGpzb24", "limit=2&cursor=WyJBVVRIIiwiQVVUSC1MT0dJTiJd", "limit=500",
	} {
		rec := httptest.NewRecorder()
		listRequirements(rec, httptest.NewRequest("GET", "/api/projects/p/requirements?"+query, nil))
		fmt.Println(query, "->", rec.Code, rec.Body)
	}
}
