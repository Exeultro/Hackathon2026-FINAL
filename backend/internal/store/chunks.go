package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Chunk struct {
	Start     float64
	End       float64
	Text      string
	Embedding []float32
}

func (s *pgStore) SaveChunks(ctx context.Context, lectureID string, chunks []Chunk, vectors [][]float32) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM chunks WHERE lecture_id = $1`, lectureID); err != nil {
		return err
	}
	for i, c := range chunks {
		if i >= len(vectors) {
			return fmt.Errorf("vectors mismatch: %d chunks, %d vectors", len(chunks), len(vectors))
		}
		vec := vectorLiteral(vectors[i])
		if _, err := tx.Exec(ctx, `
			INSERT INTO chunks (lecture_id, start_sec, end_sec, text, embedding)
			VALUES ($1, $2, $3, $4, $5::vector)`,
			lectureID, c.Start, c.End, c.Text, vec); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func vectorLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = fmt.Sprintf("%f", x)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func (s *pgStore) SaveSummary(ctx context.Context, lectureID, summary string, outline, keyTerms any) error {
	outlineJSON, err := json.Marshal(outline)
	if err != nil {
		return err
	}
	termsJSON, err := json.Marshal(keyTerms)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO summaries (lecture_id, summary, outline, key_terms)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (lecture_id) DO UPDATE
			SET summary = EXCLUDED.summary,
			    outline = EXCLUDED.outline,
			    key_terms = EXCLUDED.key_terms`,
		lectureID, summary, outlineJSON, termsJSON)
	return err
}
