package mlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Client struct {
	base string
	http *http.Client
}

func New(base string) *Client {
	return &Client{
		base: base,
		// ASR of a long lecture and the map-reduce summary on CPU can take well over an hour.
		http: &http.Client{Timeout: 4 * time.Hour},
	}
}

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type ASRResponse struct {
	Language string    `json:"language"`
	Duration float64   `json:"duration"`
	Segments []Segment `json:"segments"`
}

func (c *Client) ASR(ctx context.Context, audioPath, language string) (*ASRResponse, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("open audio: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("audio", filepath.Base(audioPath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return nil, err
	}
	if language != "" {
		_ = mw.WriteField("language", language)
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/asr", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("asr http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("asr status %d: %s", resp.StatusCode, string(b))
	}
	var out ASRResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("asr decode: %w", err)
	}
	return &out, nil
}

type EmbedResponse struct {
	Vectors [][]float32 `json:"vectors"`
}

func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	body, _ := json.Marshal(map[string]any{"texts": texts})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("embed status %d: %s", resp.StatusCode, string(b))
	}
	var out EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("embed decode: %w", err)
	}
	return out.Vectors, nil
}

type SummaryResponse struct {
	Summary  string `json:"summary"`
	Outline  []any  `json:"outline"`
	KeyTerms []any  `json:"key_terms"`
}

func (c *Client) Summary(ctx context.Context, segments []Segment) (*SummaryResponse, error) {
	body, _ := json.Marshal(map[string]any{"segments": segments})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/summary", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("summary http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("summary status %d: %s", resp.StatusCode, string(b))
	}
	var out SummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("summary decode: %w", err)
	}
	return &out, nil
}

type AskFragment struct {
	LectureTitle string  `json:"lecture_title"`
	Start        float64 `json:"start"`
	End          float64 `json:"end"`
	Text         string  `json:"text"`
}

type AskResponse struct {
	Answer string `json:"answer"`
}

func (c *Client) Ask(ctx context.Context, question string, fragments []AskFragment, history []map[string]string) (*AskResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"question":  question,
		"fragments": fragments,
		"history":   history,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/ask", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ask http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("ask status %d: %s", resp.StatusCode, string(b))
	}
	var out AskResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ask decode: %w", err)
	}
	return &out, nil
}

func (c *Client) Health(ctx context.Context) (map[string]string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/health", nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
