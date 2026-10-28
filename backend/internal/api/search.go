package api

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"aba/internal/mlclient"
	"aba/internal/store"
)

func Search(s store.Store, ml *mlclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		q := strings.TrimSpace(query.Get("q"))
		if n := utf8.RuneCountInString(q); n < 2 || n > 500 {
			writeValidation(w, map[string]interface{}{"q": "must be 2..500 chars"})
			return
		}
		limit := 10
		if v := query.Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 50 {
				writeValidation(w, map[string]interface{}{"limit": "must be 1..50"})
				return
			}
			limit = n
		}
		var lectureID, courseID *string
		if v := query.Get("lecture_id"); v != "" {
			if _, ok := lectureFor(w, r, s, v); !ok {
				return
			}
			lectureID = &v
		}
		if v := query.Get("course_id"); v != "" {
			if !courseVisible(r, s, v) {
				writeError(w, http.StatusNotFound, "not_found", "course not found")
				return
			}
			courseID = &v
		}
		start := time.Now()

		vecs, err := ml.Embed(r.Context(), []string{"query: " + q})
		if err != nil {
			writeError(w, http.StatusBadGateway, "ml_unavailable", err.Error())
			return
		}

		hits, err := s.SearchChunks(r.Context(), vecs[0], limit, lectureID, courseID, viewer(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		out := make([]SearchHit, 0, len(hits))
		for _, h := range hits {
			out = append(out, toAPISearchHit(h, q))
		}
		writeJSON(w, http.StatusOK, SearchResponse{
			Query:  q,
			TookMs: time.Since(start).Milliseconds(),
			Hits:   out,
		})
	}
}

func toAPISearchHit(h store.SearchHit, query string) SearchHit {
	text := strings.TrimSpace(h.Text)
	return SearchHit{
		Lecture: LectureRef{
			ID:           h.LectureID,
			Title:        h.LectureTitle,
			ThumbnailURL: thumbnailURL(h.LectureID, h.ThumbnailPath),
		},
		Start:      h.Start,
		End:        h.End,
		Text:       text,
		Score:      math.Max(0, math.Min(1, h.Score)),
		Highlights: highlights(text, query),
		DeepLink:   "/lectures/" + h.LectureID + "?t=" + itoaF(h.Start),
	}
}

func itoaF(f float64) string {
	return strconv.FormatInt(int64(f), 10)
}

// highlights marks words of text that share a stem with a query word, as UTF-16 [from, to) pairs.
func highlights(text, query string) [][]int {
	var stems []string
	for _, w := range strings.FieldsFunc(strings.ToLower(query), notWordRune) {
		rs := []rune(w)
		if len(rs) < 3 {
			continue
		}
		n := len(rs) - 2
		if n < 3 {
			n = 3
		}
		if n > len(rs) {
			n = len(rs)
		}
		stems = append(stems, string(rs[:n]))
	}
	out := [][]int{}
	if len(stems) == 0 {
		return out
	}
	pos := 0
	wordStart := -1
	var word []rune
	flush := func() {
		if wordStart < 0 {
			return
		}
		lw := strings.ToLower(string(word))
		for _, st := range stems {
			if strings.HasPrefix(lw, st) {
				out = append(out, []int{wordStart, pos})
				break
			}
		}
		wordStart = -1
		word = word[:0]
	}
	for _, r := range text {
		if notWordRune(r) {
			flush()
		} else {
			if wordStart < 0 {
				wordStart = pos
			}
			word = append(word, r)
		}
		pos += utf16.RuneLen(r)
	}
	flush()
	return out
}

func notWordRune(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}
