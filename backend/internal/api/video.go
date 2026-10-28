package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"aba/internal/storage"
	"aba/internal/store"
)

func GetLectureVideo(s store.Store, files *storage.Files) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "lecture_id")
		lec, err := s.GetLecture(r.Context(), id)
		if err != nil || !mediaAllowed(r, lec) {
			writeError(w, http.StatusNotFound, "not_found", "lecture not found")
			return
		}
		if lec.VideoPath == nil || *lec.VideoPath == "" {
			writeError(w, http.StatusNotFound, "not_found", "video not ready")
			return
		}

		f, err := files.Open(*lec.VideoPath)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "video file missing")
			return
		}
		defer f.Close()

		// Content-Type по расширению; ServeContent сам разберётся с Range.
		ext := strings.ToLower(filepath.Ext(*lec.VideoPath))
		switch ext {
		case ".mp4":
			w.Header().Set("Content-Type", "video/mp4")
		case ".webm":
			w.Header().Set("Content-Type", "video/webm")
		}

		stat, err := f.Stat()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		// ServeContent умеет Range, 206, Accept-Ranges, If-Modified-Since и т.д.
		http.ServeContent(w, r, filepath.Base(*lec.VideoPath), stat.ModTime(), f)
	}
}

func GetLectureThumbnail(s store.Store, files *storage.Files) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "lecture_id")
		lec, err := s.GetLecture(r.Context(), id)
		if err != nil || !mediaAllowed(r, lec) || lec.ThumbnailPath == nil || *lec.ThumbnailPath == "" {
			writeError(w, http.StatusNotFound, "not_found", "thumbnail not found")
			return
		}
		f, err := files.Open(*lec.ThumbnailPath)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "thumbnail file missing")
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "image/jpeg")
		http.ServeContent(w, r, "thumb.jpg", lec.CreatedAt, f)
	}
}
