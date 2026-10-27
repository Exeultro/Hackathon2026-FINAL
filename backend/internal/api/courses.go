package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"aba/internal/store"
)

func ListCoursesHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := s.ListCourses(r.Context(), viewer(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		out := make([]Course, 0, len(list))
		for _, c := range list {
			out = append(out, Course{
				ID: c.ID, Title: c.Title, LectureCount: c.LectureCount, Mine: mine(c.Owner, r), CreatedAt: c.CreatedAt,
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func CreateCourseHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in CourseInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Title == "" {
			writeValidation(w, map[string]interface{}{"title": "required"})
			return
		}
		c, err := s.CreateCourse(r.Context(), in.Title, viewer(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, Course{ID: c.ID, Title: c.Title, LectureCount: c.LectureCount, Mine: mine(c.Owner, r), CreatedAt: c.CreatedAt})
	}
}

func UpdateCourseHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "course_id")
		if !ownCourse(w, r, s, id) {
			return
		}
		var in CourseInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Title == "" {
			writeValidation(w, map[string]interface{}{"title": "required"})
			return
		}
		c, err := s.UpdateCourse(r.Context(), id, in.Title)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "course not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, Course{ID: c.ID, Title: c.Title, LectureCount: c.LectureCount, Mine: mine(c.Owner, r), CreatedAt: c.CreatedAt})
	}
}

func DeleteCourseHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "course_id")
		if !ownCourse(w, r, s, id) {
			return
		}
		err := s.DeleteCourse(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "course not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
