package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"aba/internal/store"
)

func ListBookmarksHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cid := owner(w, r)
		if cid == "" {
			return
		}
		var lectureID *string
		if v := r.URL.Query().Get("lecture_id"); v != "" {
			lectureID = &v
		}
		list, err := s.ListBookmarks(r.Context(), cid, lectureID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		out := make([]Bookmark, 0, len(list))
		for _, b := range list {
			out = append(out, Bookmark{
				ID: b.ID, LectureID: b.LectureID, LectureTitle: b.LectureTitle,
				T: b.T, Note: b.Note, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func CreateBookmarkHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cid := owner(w, r)
		if cid == "" {
			return
		}
		var in BookmarkCreate
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.LectureID == "" {
			writeValidation(w, map[string]interface{}{"lecture_id": "required"})
			return
		}
		if len(in.Note) > 5000 {
			writeValidation(w, map[string]interface{}{"note": "max 5000 chars"})
			return
		}
		if _, ok := lectureFor(w, r, s, in.LectureID); !ok {
			return
		}
		b, err := s.CreateBookmark(r.Context(), &store.Bookmark{
			ID:        store.NewID("bm_"),
			ClientID:  cid,
			LectureID: in.LectureID,
			T:         in.T,
			Note:      in.Note,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, Bookmark{
			ID: b.ID, LectureID: b.LectureID, LectureTitle: b.LectureTitle,
			T: b.T, Note: b.Note, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
		})
	}
}

func UpdateBookmarkHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cid := owner(w, r)
		if cid == "" {
			return
		}
		id := chi.URLParam(r, "bookmark_id")
		var in struct {
			Note *string  `json:"note"`
			T    *float64 `json:"t"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeValidation(w, map[string]interface{}{"body": "invalid"})
			return
		}
		b, err := s.UpdateBookmark(r.Context(), cid, id, in.Note, in.T)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "bookmark not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, Bookmark{
			ID: b.ID, LectureID: b.LectureID, LectureTitle: b.LectureTitle,
			T: b.T, Note: b.Note, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
		})
	}
}

func DeleteBookmarkHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cid := owner(w, r)
		if cid == "" {
			return
		}
		id := chi.URLParam(r, "bookmark_id")
		err := s.DeleteBookmark(r.Context(), cid, id)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "bookmark not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func ListProgressHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cid := owner(w, r)
		if cid == "" {
			return
		}
		list, err := s.ListProgress(r.Context(), cid)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		out := make([]WatchProgress, 0, len(list))
		for _, p := range list {
			out = append(out, WatchProgress{LectureID: p.LectureID, Position: p.Position, UpdatedAt: p.UpdatedAt})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func ClearProgressHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cid := owner(w, r)
		if cid == "" {
			return
		}
		if err := s.ClearProgress(r.Context(), cid); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func SaveProgressHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cid := owner(w, r)
		if cid == "" {
			return
		}
		lectureID := chi.URLParam(r, "lecture_id")
		var in struct {
			Position float64 `json:"position"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeValidation(w, map[string]interface{}{"body": "invalid"})
			return
		}
		if in.Position < 0 {
			writeValidation(w, map[string]interface{}{"position": "must be >= 0"})
			return
		}
		if _, ok := lectureFor(w, r, s, lectureID); !ok {
			return
		}
		p, err := s.SaveProgress(r.Context(), cid, lectureID, in.Position)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, WatchProgress{
			LectureID: p.LectureID, Position: p.Position, UpdatedAt: p.UpdatedAt,
		})
	}
}
