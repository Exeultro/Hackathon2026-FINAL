package store

import (
	"context"
)

type Segment struct {
	Idx   int
	Start float64
	End   float64
	Text  string
}

func (s *pgStore) SaveSegments(ctx context.Context, lectureID string, segs []Segment) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM segments WHERE lecture_id = $1`, lectureID); err != nil {
		return err
	}
	for _, seg := range segs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO segments (lecture_id, idx, start_sec, end_sec, text)
			VALUES ($1, $2, $3, $4, $5)`,
			lectureID, seg.Idx, seg.Start, seg.End, seg.Text); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *pgStore) ListSegments(ctx context.Context, lectureID string) ([]Segment, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT idx, start_sec, end_sec, text
		FROM segments WHERE lecture_id = $1 ORDER BY idx`, lectureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Segment{}
	for rows.Next() {
		var seg Segment
		if err := rows.Scan(&seg.Idx, &seg.Start, &seg.End, &seg.Text); err != nil {
			return nil, err
		}
		out = append(out, seg)
	}
	return out, rows.Err()
}
