package store

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

type SummaryData struct {
	Summary  string
	Outline  json.RawMessage
	KeyTerms json.RawMessage
}

func (s *pgStore) GetSummary(ctx context.Context, lectureID string) (*SummaryData, error) {
	var out SummaryData
	err := s.pool.QueryRow(ctx, `
		SELECT summary, outline, key_terms FROM summaries WHERE lecture_id = $1`,
		lectureID,
	).Scan(&out.Summary, &out.Outline, &out.KeyTerms)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}
