// Package httpapi — learning.go: per-lesson learning state endpoints.
//
// All single-user; lessonId comes from the path. One handler type per resource
// keeps the ServeMux registrations in router.go flat and greppable.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"stacklab/server/internal/learningstore"
)

// writeJSON encodes v with the application/json content type.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

type learningHandlers struct {
	store  *learningstore.Store
	logger *slog.Logger
}

func (h learningHandlers) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(v); err != nil {
		problem(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func (h learningHandlers) lessonID(r *http.Request) string {
	return r.PathValue("lessonId")
}

// GET /api/progress — every lesson's status for dashboard/sidebar badges.
func (h learningHandlers) progress(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.ListProgress(r.Context())
	if err != nil {
		h.logger.Error("list progress", slog.Any("err", err))
		problem(w, http.StatusInternalServerError, "could not load progress")
		return
	}
	if rows == nil {
		rows = []learningstore.ProgressRow{}
	}
	writeJSON(w, rows)
}

func (h learningHandlers) markViewed(w http.ResponseWriter, r *http.Request) {
	h.mark(w, r, h.store.MarkViewed)
}

func (h learningHandlers) markRan(w http.ResponseWriter, r *http.Request) {
	h.mark(w, r, h.store.MarkRan)
}

func (h learningHandlers) markCompleted(w http.ResponseWriter, r *http.Request) {
	h.mark(w, r, h.store.MarkCompleted)
}

func (h learningHandlers) mark(w http.ResponseWriter, r *http.Request, f func(context.Context, string) error) {
	if err := f(r.Context(), h.lessonID(r)); err != nil {
		h.logger.Error("mark progress", slog.Any("err", err))
		problem(w, http.StatusInternalServerError, "could not update progress")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/notes/{lessonId}
func (h learningHandlers) getNote(w http.ResponseWriter, r *http.Request) {
	body, err := h.store.GetNote(r.Context(), h.lessonID(r))
	if err != nil {
		h.logger.Error("get note", slog.Any("err", err))
		problem(w, http.StatusInternalServerError, "could not load note")
		return
	}
	writeJSON(w, map[string]string{"body": body})
}

// PUT /api/notes/{lessonId} — body {"body": "..."}
func (h learningHandlers) putNote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Body string `json:"body"`
	}
	if !h.decode(w, r, &req) {
		return
	}
	if err := h.store.SaveNote(r.Context(), h.lessonID(r), req.Body); err != nil {
		h.logger.Error("save note", slog.Any("err", err))
		problem(w, http.StatusInternalServerError, "could not save note")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/bookmarks — lesson ids, newest first.
func (h learningHandlers) listBookmarks(w http.ResponseWriter, r *http.Request) {
	ids, err := h.store.ListBookmarks(r.Context())
	if err != nil {
		h.logger.Error("list bookmarks", slog.Any("err", err))
		problem(w, http.StatusInternalServerError, "could not load bookmarks")
		return
	}
	if ids == nil {
		ids = []string{}
	}
	writeJSON(w, ids)
}

func (h learningHandlers) addBookmark(w http.ResponseWriter, r *http.Request) {
	if err := h.store.AddBookmark(r.Context(), h.lessonID(r)); err != nil {
		h.logger.Error("add bookmark", slog.Any("err", err))
		problem(w, http.StatusInternalServerError, "could not add bookmark")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h learningHandlers) removeBookmark(w http.ResponseWriter, r *http.Request) {
	if err := h.store.RemoveBookmark(r.Context(), h.lessonID(r)); err != nil {
		h.logger.Error("remove bookmark", slog.Any("err", err))
		problem(w, http.StatusInternalServerError, "could not remove bookmark")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/quiz/{lessonId} — {questionIndex: optionIndex}
func (h learningHandlers) getQuiz(w http.ResponseWriter, r *http.Request) {
	answers, err := h.store.GetQuizAnswers(r.Context(), h.lessonID(r))
	if err != nil {
		h.logger.Error("get quiz answers", slog.Any("err", err))
		problem(w, http.StatusInternalServerError, "could not load quiz answers")
		return
	}
	writeJSON(w, answers)
}

// PUT /api/quiz/{lessonId} — {questionIndex: optionIndex}
func (h learningHandlers) putQuiz(w http.ResponseWriter, r *http.Request) {
	var req map[string]int
	if !h.decode(w, r, &req) {
		return
	}
	if err := h.store.SaveQuizAnswers(r.Context(), h.lessonID(r), req); err != nil {
		h.logger.Error("save quiz answers", slog.Any("err", err))
		problem(w, http.StatusInternalServerError, "could not save quiz answers")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/history?lesson={id} — recent runs, optionally scoped to one lesson.
func (h learningHandlers) history(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.History(r.Context(), r.URL.Query().Get("lesson"))
	if err != nil {
		h.logger.Error("run history", slog.Any("err", err))
		problem(w, http.StatusInternalServerError, "could not load run history")
		return
	}
	if rows == nil {
		rows = []learningstore.RunRow{}
	}
	writeJSON(w, rows)
}
