package store

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5"
)

const lectureCols = `id, title, course_id, status, progress, error,
	summary_status, source, source_url,
	playback_type, playback_url, provider_id,
	duration, language, video_path, audio_path, thumbnail_path,
	owner, created_at, updated_at`

func scanLecture(row pgx.Row) (*Lecture, error) {
	var l Lecture
	err := row.Scan(
		&l.ID, &l.Title, &l.CourseID, &l.Status, &l.Progress, &l.Error,
		&l.SummaryStatus, &l.Source, &l.SourceURL,
		&l.PlaybackType, &l.PlaybackURL, &l.ProviderID,
		&l.Duration, &l.Language, &l.VideoPath, &l.AudioPath, &l.ThumbnailPath,
		&l.Owner, &l.CreatedAt, &l.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *pgStore) ListLectures(ctx context.Context, courseID *string, status *string, viewer string) ([]Lecture, error) {
	args := []any{viewer}
	q := `SELECT ` + lectureCols + ` FROM lectures WHERE (owner IS NULL OR owner = $1)`
	if courseID != nil {
		args = append(args, *courseID)
		q += ` AND course_id = $` + strconv.Itoa(len(args))
	}
	if status != nil {
		args = append(args, *status)
		q += ` AND status = $` + strconv.Itoa(len(args))
	}
	q += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Lecture{}
	for rows.Next() {
		l, err := scanLecture(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

func (s *pgStore) GetLecture(ctx context.Context, id string) (*Lecture, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+lectureCols+` FROM lectures WHERE id = $1`, id)
	return scanLecture(row)
}

func (s *pgStore) CreateLecture(ctx context.Context, l *Lecture) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO lectures (id, title, course_id, status, progress, summary_status, source, source_url, language, owner, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,now(),now())`,
		l.ID, l.Title, l.CourseID, l.Status, l.Progress, l.SummaryStatus, l.Source, l.SourceURL, l.Language, l.Owner,
	)
	return err
}

func (s *pgStore) UpdateLecture(ctx context.Context, id string, title *string, setCourse bool, courseID *string) (*Lecture, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE lectures SET
			title = COALESCE($2, title),
			course_id = CASE WHEN $4 THEN $3 ELSE course_id END,
			updated_at = now()
		WHERE id = $1
		RETURNING `+lectureCols,
		id, title, courseID, setCourse,
	)
	return scanLecture(row)
}

func (s *pgStore) DeleteLecture(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM lectures WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *pgStore) SetLectureStatus(ctx context.Context, id string, status LectureStatus, progress float64, errMsg *string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE lectures SET status = $2, progress = $3, error = $4, updated_at = now()
		WHERE id = $1`, id, status, progress, errMsg)
	return err
}

func (s *pgStore) SetSummaryStatus(ctx context.Context, id string, status SummaryStatus, errMsg *string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE lectures SET summary_status = $2, error = $3, updated_at = now()
		WHERE id = $1`, id, status, errMsg)
	return err
}

func (s *pgStore) SetLectureVideo(ctx context.Context, id, videoPath string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE lectures SET video_path = $2, updated_at = now() WHERE id = $1`,
		id, videoPath)
	return err
}

func (s *pgStore) SetDurationAndLanguage(ctx context.Context, id string, duration float64, language string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE lectures SET duration = $2, language = $3, updated_at = now() WHERE id = $1`,
		id, duration, language)
	return err
}

func (s *pgStore) SetDuration(ctx context.Context, id string, duration float64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE lectures SET duration = $2, updated_at = now() WHERE id = $1`,
		id, duration)
	return err
}

func (s *pgStore) SetLectureThumbnail(ctx context.Context, id, thumbPath string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE lectures SET thumbnail_path = $2, updated_at = now() WHERE id = $1`,
		id, thumbPath)
	return err
}
