package api

import (
	"encoding/json"
	"net/http"
	"unicode/utf8"

	"aba/internal/mlclient"
	"aba/internal/store"
)

func Ask(s store.Store, ml *mlclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeValidation(w, map[string]interface{}{"body": "invalid json"})
			return
		}
		if n := utf8.RuneCountInString(req.Question); n < 2 || n > 1000 {
			writeValidation(w, map[string]interface{}{"question": "must be 2..1000 chars"})
			return
		}

		if req.LectureID != nil {
			if _, ok := lectureFor(w, r, s, *req.LectureID); !ok {
				return
			}
		}
		if req.CourseID != nil && !courseVisible(r, s, *req.CourseID) {
			writeError(w, http.StatusNotFound, "not_found", "course not found")
			return
		}

		vecs, err := ml.Embed(r.Context(), []string{"query: " + req.Question})
		if err != nil {
			writeError(w, http.StatusBadGateway, "ml_unavailable", err.Error())
			return
		}
		hits, err := s.SearchChunks(r.Context(), vecs[0], 5, req.LectureID, req.CourseID, viewer(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		fragments := make([]mlclient.AskFragment, len(hits))
		for i, h := range hits {
			fragments[i] = mlclient.AskFragment{
				LectureTitle: h.LectureTitle,
				Start:        h.Start,
				End:          h.End,
				Text:         h.Text,
			}
		}

		if len(req.History) > 20 {
			req.History = req.History[len(req.History)-20:]
		}
		var history []map[string]string
		for _, m := range req.History {
			history = append(history, map[string]string{"role": m.Role, "content": m.Content})
		}
		if history == nil {
			history = []map[string]string{}
		}
		answer, err := ml.Ask(r.Context(), req.Question, fragments, history)
		if err != nil {
			writeError(w, http.StatusBadGateway, "ml_unavailable", err.Error())
			return
		}

		sources := make([]SearchHit, 0, len(hits))
		for _, h := range hits {
			sources = append(sources, toAPISearchHit(h, req.Question))
		}

		writeJSON(w, http.StatusOK, AskResponse{
			Answer:  answer.Answer,
			Sources: sources,
		})
	}
}
