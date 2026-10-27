package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"aba/internal/store"
)

func GetTranscript(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "lecture_id")
		lec, ok := lectureFor(w, r, s, id)
		if !ok {
			return
		}
		if lec.Status != store.StatusReady {
			writeError(w, http.StatusConflict, "not_ready", "lecture is not ready")
			return
		}
		segs, err := s.ListSegments(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		lang := "ru"
		if lec.Language != nil {
			lang = *lec.Language
		}
		out := Transcript{
			LectureID: id,
			Language:  lang,
			Segments:  make([]Segment, 0, len(segs)),
		}
		for _, seg := range segs {
			out.Segments = append(out.Segments, Segment{
				ID: seg.Idx, Start: seg.Start, End: seg.End, Text: seg.Text,
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func GetSubtitles(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "lecture_id")
		lec, ok := lectureFor(w, r, s, id)
		if !ok {
			return
		}
		if lec.Status != store.StatusReady {
			writeError(w, http.StatusConflict, "not_ready", "lecture is not ready")
			return
		}
		segs, err := s.ListSegments(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
		fmt.Fprintln(w, "WEBVTT")
		fmt.Fprintln(w)
		for _, seg := range segs {
			fmt.Fprintf(w, "%d\n", seg.Idx+1)
			fmt.Fprintf(w, "%s --> %s\n", formatVTTTime(seg.Start), formatVTTTime(seg.End))
			fmt.Fprintln(w, seg.Text)
			fmt.Fprintln(w)
		}
	}
}

func GetSummary(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "lecture_id")
		lec, ok := lectureFor(w, r, s, id)
		if !ok {
			return
		}
		if lec.SummaryStatus != store.SummaryReady {
			writeError(w, http.StatusConflict, "not_ready", "summary is not ready")
			return
		}
		data, err := s.GetSummary(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		out := map[string]any{
			"lecture_id": id,
			"summary":    data.Summary,
			"outline":    json.RawMessage(data.Outline),
			"key_terms":  json.RawMessage(data.KeyTerms),
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func formatVTTTime(sec float64) string {
	ms := int64(sec * 1000.0)
	h := ms / 3600000
	m := (ms % 3600000) / 60000
	s := (ms % 60000) / 1000
	millis := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, millis)
}
