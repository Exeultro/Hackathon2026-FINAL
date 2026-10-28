package store

import (
	"context"
	"strconv"
)

type SearchHit struct {
	LectureID     string
	LectureTitle  string
	Start         float64
	End           float64
	Text          string
	Score         float64
	ThumbnailPath *string
}

// Ordering by the score expression keeps the scan exact: an HNSW scan with a WHERE filter can return fewer rows than LIMIT.
func (s *pgStore) SearchChunks(ctx context.Context, vec []float32, limit int, lectureID, courseID *string, viewer string) ([]SearchHit, error) {
	q := `
		SELECT c.lecture_id, l.title, c.start_sec, c.end_sec, c.text,
		       1 - (c.embedding <=> $1::vector) AS score, l.thumbnail_path
		FROM chunks c
		JOIN lectures l ON l.id = c.lecture_id
		WHERE l.status = 'ready' AND (l.owner IS NULL OR l.owner = $2)`
	args := []any{vectorLiteral(vec), viewer}
	if lectureID != nil {
		args = append(args, *lectureID)
		q += ` AND c.lecture_id = $` + strconv.Itoa(len(args))
	}
	if courseID != nil {
		args = append(args, *courseID)
		q += ` AND l.course_id = $` + strconv.Itoa(len(args))
	}
	args = append(args, limit)
	q += ` ORDER BY score DESC LIMIT $` + strconv.Itoa(len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []SearchHit{}
	for rows.Next() {
		var h SearchHit
		if err := rows.Scan(&h.LectureID, &h.LectureTitle, &h.Start, &h.End, &h.Text, &h.Score, &h.ThumbnailPath); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
