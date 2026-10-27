package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Bookmark struct {
	ID           string
	ClientID     string
	LectureID    string
	LectureTitle string
	T            float64
	Note         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *pgStore) ListBookmarks(ctx context.Context, clientID string, lectureID *string) ([]Bookmark, error) {
	q := `
		SELECT b.id, b.client_id, b.lecture_id, l.title, b.t, b.note, b.created_at, b.updated_at
		FROM bookmarks b
		JOIN lectures l ON l.id = b.lecture_id
		WHERE b.client_id = $1`
	args := []any{clientID}
	if lectureID != nil {
		args = append(args, *lectureID)
		q += ` AND b.lecture_id = $2`
	}
	q += ` ORDER BY b.lecture_id, b.t`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Bookmark{}
	for rows.Next() {
		var b Bookmark
		if err := rows.Scan(&b.ID, &b.ClientID, &b.LectureID, &b.LectureTitle, &b.T, &b.Note, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *pgStore) CreateBookmark(ctx context.Context, b *Bookmark) (*Bookmark, error) {
	var out Bookmark
	err := s.pool.QueryRow(ctx, `
		INSERT INTO bookmarks (id, client_id, user_id, lecture_id, t, note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now(), now())
		RETURNING id, client_id, lecture_id, t, note, created_at, updated_at`,
		b.ID, b.ClientID, ownerUserID(b.ClientID), b.LectureID, b.T, b.Note,
	).Scan(&out.ID, &out.ClientID, &out.LectureID, &out.T, &out.Note, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = s.pool.QueryRow(ctx, `SELECT title FROM lectures WHERE id = $1`, out.LectureID).Scan(&out.LectureTitle)
	return &out, nil
}

func (s *pgStore) UpdateBookmark(ctx context.Context, clientID, id string, note *string, t *float64) (*Bookmark, error) {
	var out Bookmark
	err := s.pool.QueryRow(ctx, `
		UPDATE bookmarks SET note = COALESCE($3, note), t = COALESCE($4, t), updated_at = now()
		WHERE id = $1 AND client_id = $2
		RETURNING id, client_id, lecture_id, t, note, created_at, updated_at`,
		id, clientID, note, t,
	).Scan(&out.ID, &out.ClientID, &out.LectureID, &out.T, &out.Note, &out.CreatedAt, &out.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = s.pool.QueryRow(ctx, `SELECT title FROM lectures WHERE id = $1`, out.LectureID).Scan(&out.LectureTitle)
	return &out, nil
}

func (s *pgStore) DeleteBookmark(ctx context.Context, clientID, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM bookmarks WHERE id = $1 AND client_id = $2`, id, clientID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
