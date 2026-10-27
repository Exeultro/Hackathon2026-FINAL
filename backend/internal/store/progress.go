package store

import (
	"context"
	"time"
)

type WatchProgress struct {
	LectureID string
	Position  float64
	UpdatedAt time.Time
}

func (s *pgStore) ListProgress(ctx context.Context, clientID string) ([]WatchProgress, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT lecture_id, position, updated_at
		FROM watch_progress
		WHERE client_id = $1
		ORDER BY updated_at DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []WatchProgress{}
	for rows.Next() {
		var p WatchProgress
		if err := rows.Scan(&p.LectureID, &p.Position, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *pgStore) SaveProgress(ctx context.Context, clientID, lectureID string, position float64) (*WatchProgress, error) {
	var p WatchProgress
	err := s.pool.QueryRow(ctx, `
		INSERT INTO watch_progress (client_id, user_id, lecture_id, position, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (client_id, lecture_id) DO UPDATE
			SET position = EXCLUDED.position, updated_at = now()
		RETURNING lecture_id, position, updated_at`,
		clientID, ownerUserID(clientID), lectureID, position,
	).Scan(&p.LectureID, &p.Position, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *pgStore) ClearProgress(ctx context.Context, clientID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM watch_progress WHERE client_id = $1`, clientID)
	return err
}
