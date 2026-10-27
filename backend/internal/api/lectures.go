package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"aba/internal/pipeline"
	"aba/internal/storage"
	"aba/internal/store"
)

func ListLecturesHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var courseID, status *string
		if v := r.URL.Query().Get("course_id"); v != "" {
			courseID = &v
		}
		if v := r.URL.Query().Get("status"); v != "" {
			status = &v
		}
		list, err := s.ListLectures(r.Context(), courseID, status, viewer(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		out := make([]Lecture, 0, len(list))
		for _, l := range list {
			out = append(out, toAPILecture(l, viewer(r)))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func UploadLectureHandler(s store.Store, files *storage.Files, pipe *pipeline.Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, storage.MaxUploadBytes)

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "max 2 GB")
				return
			}
			writeValidation(w, map[string]interface{}{"body": err.Error()})
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			writeValidation(w, map[string]interface{}{"file": "required"})
			return
		}
		defer file.Close()

		title := r.FormValue("title")
		if title == "" {
			base := header.Filename
			title = strings.TrimSuffix(base, filepath.Ext(base))
		}
		courseID := r.FormValue("course_id")
		var coursePtr *string
		if courseID != "" {
			if !courseVisible(r, s, courseID) {
				writeValidation(w, map[string]interface{}{"course_id": "course not found"})
				return
			}
			coursePtr = &courseID
		}
		owner := viewer(r)

		lec := &store.Lecture{
			ID:            store.NewID("lec_"),
			Title:         title,
			CourseID:      coursePtr,
			Status:        store.StatusQueued,
			Progress:      0,
			SummaryStatus: store.SummaryPending,
			Source:        "upload",
			Language:      languageHint(r.FormValue("language")),
			Owner:         &owner,
		}
		if err := s.CreateLecture(r.Context(), lec); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		videoPath, err := files.SaveVideo(lec.ID, header, file)
		if err != nil {
			_ = s.DeleteLecture(r.Context(), lec.ID)
			if errors.Is(err, storage.ErrUnsupported) {
				writeError(w, http.StatusUnsupportedMediaType, "unsupported_format", "only mp4/webm")
				return
			}
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "max 2 GB")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		if err := s.SetLectureVideo(r.Context(), lec.ID, videoPath); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		pipe.Enqueue(lec.ID)

		got, err := s.GetLecture(r.Context(), lec.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, toAPILecture(*got, viewer(r)))
	}
}

func ImportLectureHandler(s store.Store, pipe *pipeline.Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			URL      string  `json:"url"`
			Title    *string `json:"title"`
			CourseID *string `json:"course_id"`
			Language string  `json:"language"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.URL == "" {
			writeValidation(w, map[string]interface{}{"url": "required"})
			return
		}
		u, err := url.Parse(strings.TrimSpace(in.URL))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			writeValidation(w, map[string]interface{}{"url": "must be an http(s) link"})
			return
		}
		in.URL = u.String()
		title := in.URL
		if in.Title != nil && strings.TrimSpace(*in.Title) != "" {
			title = strings.TrimSpace(*in.Title)
		}
		if in.CourseID != nil && *in.CourseID == "" {
			in.CourseID = nil
		}
		if in.CourseID != nil && !courseVisible(r, s, *in.CourseID) {
			writeValidation(w, map[string]interface{}{"course_id": "course not found"})
			return
		}
		owner := viewer(r)
		if owner == "" {
			writeError(w, http.StatusBadRequest, "client_id_required", "X-Client-Id header is required")
			return
		}
		lec := &store.Lecture{
			ID:            store.NewID("lec_"),
			Title:         title,
			CourseID:      in.CourseID,
			Status:        store.StatusQueued,
			Progress:      0,
			SummaryStatus: store.SummaryPending,
			Source:        sourceOf(u.Hostname()),
			SourceURL:     &in.URL,
			Language:      languageHint(in.Language),
			Owner:         &owner,
		}
		if err := s.CreateLecture(r.Context(), lec); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		pipe.Enqueue(lec.ID)

		got, err := s.GetLecture(r.Context(), lec.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, toAPILecture(*got, viewer(r)))
	}
}

func GetLectureHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l, ok := lectureFor(w, r, s, chi.URLParam(r, "lecture_id"))
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, toAPILecture(*l, viewer(r)))
	}
}

func UpdateLectureHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "lecture_id")
		if _, ok := ownLecture(w, r, s, id); !ok {
			return
		}
		var in map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeValidation(w, map[string]interface{}{"body": "invalid"})
			return
		}
		var title, courseID *string
		if raw, ok := in["title"]; ok {
			if err := json.Unmarshal(raw, &title); err != nil || title == nil || strings.TrimSpace(*title) == "" {
				writeValidation(w, map[string]interface{}{"title": "must be a non-empty string"})
				return
			}
		}
		rawCourse, setCourse := in["course_id"]
		if setCourse {
			if err := json.Unmarshal(rawCourse, &courseID); err != nil {
				writeValidation(w, map[string]interface{}{"course_id": "must be a string or null"})
				return
			}
			if courseID != nil && *courseID == "" {
				courseID = nil
			}
			if courseID != nil && !courseVisible(r, s, *courseID) {
				writeValidation(w, map[string]interface{}{"course_id": "course not found"})
				return
			}
		}
		l, err := s.UpdateLecture(r.Context(), id, title, setCourse, courseID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "lecture not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toAPILecture(*l, viewer(r)))
	}
}

func DeleteLectureHandler(s store.Store, files *storage.Files) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "lecture_id")
		lec, ok := ownLecture(w, r, s, id)
		if !ok {
			return
		}
		if lec.VideoPath != nil {
			_ = files.Delete(*lec.VideoPath)
		}
		if lec.AudioPath != nil {
			_ = files.Delete(*lec.AudioPath)
		}
		if lec.ThumbnailPath != nil {
			_ = files.Delete(*lec.ThumbnailPath)
		}
		if err := s.DeleteLecture(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func RetryLectureHandler(s store.Store, pipe *pipeline.Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "lecture_id")
		l, ok := lectureFor(w, r, s, id)
		if !ok {
			return
		}
		switch {
		case l.Status == store.StatusFailed:
			if err := s.SetLectureStatus(r.Context(), id, store.StatusQueued, 0, nil); err != nil {
				writeError(w, http.StatusInternalServerError, "internal", err.Error())
				return
			}
			pipe.Enqueue(id)
		case l.Status == store.StatusReady && l.SummaryStatus == store.SummaryFailed:
			if err := pipe.RetrySummary(id); err != nil {
				writeError(w, http.StatusInternalServerError, "internal", err.Error())
				return
			}
		default:
			writeError(w, http.StatusConflict, "not_failed", "lecture is not failed")
			return
		}
		got, err := s.GetLecture(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, toAPILecture(*got, viewer(r)))
	}
}

func toAPILecture(l store.Lecture, viewer string) Lecture {
	var playback *Playback
	if l.VideoPath != nil && *l.VideoPath != "" {
		playback = &Playback{
			Type: "file",
			URL:  signedMediaURL(l.ID, "video"),
		}
	} else if l.PlaybackURL != nil && *l.PlaybackURL != "" {
		t := "file"
		if l.PlaybackType != nil {
			t = *l.PlaybackType
		}
		playback = &Playback{
			Type:       t,
			URL:        *l.PlaybackURL,
			ProviderID: l.ProviderID,
		}
	}

	return Lecture{
		ID:            l.ID,
		Title:         l.Title,
		CourseID:      l.CourseID,
		Status:        LectureStatus(l.Status),
		Progress:      l.Progress,
		Error:         l.Error,
		SummaryStatus: SummaryStatus(l.SummaryStatus),
		Source:        l.Source,
		SourceURL:     l.SourceURL,
		Playback:      playback,
		Duration:      l.Duration,
		Language:      l.Language,
		ThumbnailURL:  thumbnailURL(l.ID, l.ThumbnailPath),
		Mine:          l.Owner != nil && *l.Owner == viewer,
		CreatedAt:     l.CreatedAt,
	}
}

// PublicURL is the absolute origin of this API as the browser sees it; playback and thumbnail URLs are built from it.
var PublicURL = "http://localhost:8000"

func thumbnailURL(lectureID string, path *string) *string {
	if path == nil || *path == "" {
		return nil
	}
	u := signedMediaURL(lectureID, "thumbnail")
	return &u
}

func languageHint(v string) *string {
	if v == "" || v == "auto" {
		return nil
	}
	return &v
}

func sourceOf(host string) string {
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	switch {
	case host == "youtu.be" || host == "youtube.com" || strings.HasSuffix(host, ".youtube.com"):
		return "youtube"
	case host == "vimeo.com" || strings.HasSuffix(host, ".vimeo.com"):
		return "vimeo"
	}
	return "url"
}
