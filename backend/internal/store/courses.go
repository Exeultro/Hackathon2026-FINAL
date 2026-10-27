package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (s *pgStore) ListCourses(ctx context.Context, viewer string) ([]Course, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.title, c.owner, c.created_at,
		       (SELECT COUNT(*) FROM lectures l WHERE l.course_id = c.id
		          AND (l.owner IS NULL OR l.owner = $1)) AS lecture_count
		FROM courses c
		WHERE c.owner IS NULL OR c.owner = $1
		ORDER BY c.created_at DESC`, viewer)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Course{}
	for rows.Next() {
		var c Course
		if err := rows.Scan(&c.ID, &c.Title, &c.Owner, &c.CreatedAt, &c.LectureCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *pgStore) GetCourse(ctx context.Context, id string) (*Course, error) {
	var c Course
	err := s.pool.QueryRow(ctx, `SELECT id, title, owner, created_at FROM courses WHERE id = $1`, id).
		Scan(&c.ID, &c.Title, &c.Owner, &c.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *pgStore) CreateCourse(ctx context.Context, title, owner string) (*Course, error) {
	var c Course
	err := s.pool.QueryRow(ctx, `
		INSERT INTO courses (id, title, owner) VALUES ($1, $2, $3)
		RETURNING id, title, owner, created_at`,
		NewID("crs_"), title, owner,
	).Scan(&c.ID, &c.Title, &c.Owner, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *pgStore) UpdateCourse(ctx context.Context, id, title string) (*Course, error) {
	var c Course
	err := s.pool.QueryRow(ctx, `
		UPDATE courses SET title = $2 WHERE id = $1
		RETURNING id, title, owner, created_at`,
		id, title,
	).Scan(&c.ID, &c.Title, &c.Owner, &c.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM lectures WHERE course_id = $1`, id).Scan(&c.LectureCount)
	return &c, nil
}

func (s *pgStore) DeleteCourse(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
