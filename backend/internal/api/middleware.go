package api

import (
	"errors"
	"net/http"

	"aba/internal/store"
)

func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Client-Id")
				w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
				w.Header().Set("Access-Control-Max-Age", "300")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func viewer(r *http.Request) string {
	if id := userID(r); id != "" {
		return store.UserOwner(id)
	}
	if id := r.Header.Get("X-Client-Id"); !store.IsUserOwner(id) {
		return id
	}
	return ""
}

func lectureFor(w http.ResponseWriter, r *http.Request, s store.Store, id string) (*store.Lecture, bool) {
	l, err := s.GetLecture(r.Context(), id)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return nil, false
	}
	if err != nil || !store.Visible(l.Owner, viewer(r)) {
		writeError(w, http.StatusNotFound, "not_found", "lecture not found")
		return nil, false
	}
	return l, true
}

func mine(owner *string, r *http.Request) bool {
	v := viewer(r)
	return owner != nil && v != "" && *owner == v
}

func ownLecture(w http.ResponseWriter, r *http.Request, s store.Store, id string) (*store.Lecture, bool) {
	l, ok := lectureFor(w, r, s, id)
	if ok && !mine(l.Owner, r) {
		writeError(w, http.StatusForbidden, "forbidden", "shared lecture is read-only")
		return nil, false
	}
	return l, ok
}

func ownCourse(w http.ResponseWriter, r *http.Request, s store.Store, id string) bool {
	c, err := s.GetCourse(r.Context(), id)
	if err != nil || !store.Visible(c.Owner, viewer(r)) {
		writeError(w, http.StatusNotFound, "not_found", "course not found")
		return false
	}
	if !mine(c.Owner, r) {
		writeError(w, http.StatusForbidden, "forbidden", "shared course is read-only")
		return false
	}
	return true
}

func courseVisible(r *http.Request, s store.Store, id string) bool {
	c, err := s.GetCourse(r.Context(), id)
	return err == nil && store.Visible(c.Owner, viewer(r))
}

func owner(w http.ResponseWriter, r *http.Request) string {
	if id := userID(r); id != "" {
		return store.UserOwner(id)
	}
	id := r.Header.Get("X-Client-Id")
	if id == "" || store.IsUserOwner(id) {
		writeError(w, http.StatusBadRequest, "client_id_required", "X-Client-Id header is required")
		return ""
	}
	return id
}
