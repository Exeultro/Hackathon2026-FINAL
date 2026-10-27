package api

import "time"

type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type LectureStatus string

const (
	StatusQueued          LectureStatus = "queued"
	StatusDownloading     LectureStatus = "downloading"
	StatusExtractingAudio LectureStatus = "extracting_audio"
	StatusTranscribing    LectureStatus = "transcribing"
	StatusIndexing        LectureStatus = "indexing"
	StatusReady           LectureStatus = "ready"
	StatusFailed          LectureStatus = "failed"
)

type SummaryStatus string

const (
	SummaryPending    SummaryStatus = "pending"
	SummaryProcessing SummaryStatus = "processing"
	SummaryReady      SummaryStatus = "ready"
	SummaryFailed     SummaryStatus = "failed"
)

type Playback struct {
	Type       string  `json:"type"`
	URL        string  `json:"url"`
	ProviderID *string `json:"provider_id,omitempty"`
}

type Lecture struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	CourseID      *string       `json:"course_id,omitempty"`
	Status        LectureStatus `json:"status"`
	Progress      float64       `json:"progress"`
	Error         *string       `json:"error,omitempty"`
	SummaryStatus SummaryStatus `json:"summary_status"`
	Source        string        `json:"source"`
	SourceURL     *string       `json:"source_url,omitempty"`
	Playback      *Playback     `json:"playback,omitempty"`
	Duration      *float64      `json:"duration,omitempty"`
	Language      *string       `json:"language,omitempty"`
	ThumbnailURL  *string       `json:"thumbnail_url,omitempty"`
	Mine          bool          `json:"mine"`
	CreatedAt     time.Time     `json:"created_at"`
}

type Segment struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type Transcript struct {
	LectureID string    `json:"lecture_id"`
	Language  string    `json:"language"`
	Segments  []Segment `json:"segments"`
}

type OutlineItem struct {
	Title    string        `json:"title"`
	Start    float64       `json:"start"`
	End      float64       `json:"end"`
	Summary  *string       `json:"summary,omitempty"`
	Children []OutlineItem `json:"children"`
}

type KeyTerm struct {
	Term        string    `json:"term"`
	Definition  *string   `json:"definition,omitempty"`
	Occurrences []float64 `json:"occurrences"`
}

type Summary struct {
	LectureID string        `json:"lecture_id"`
	Summary   string        `json:"summary"`
	Outline   []OutlineItem `json:"outline"`
	KeyTerms  []KeyTerm     `json:"key_terms"`
}

type LectureRef struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	ThumbnailURL *string `json:"thumbnail_url,omitempty"`
}

type SearchHit struct {
	Lecture    LectureRef `json:"lecture"`
	Start      float64    `json:"start"`
	End        float64    `json:"end"`
	Text       string     `json:"text"`
	Score      float64    `json:"score"`
	Highlights [][]int    `json:"highlights"`
	DeepLink   string     `json:"deep_link"`
}

type SearchResponse struct {
	Query  string      `json:"query"`
	TookMs int64       `json:"took_ms"`
	Hits   []SearchHit `json:"hits"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AskRequest struct {
	Question  string        `json:"question"`
	LectureID *string       `json:"lecture_id,omitempty"`
	CourseID  *string       `json:"course_id,omitempty"`
	History   []ChatMessage `json:"history,omitempty"`
}

type AskResponse struct {
	Answer  string      `json:"answer"`
	Sources []SearchHit `json:"sources"`
}

type Course struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	LectureCount int       `json:"lecture_count"`
	Mine         bool      `json:"mine"`
	CreatedAt    time.Time `json:"created_at"`
}

type CourseInput struct {
	Title string `json:"title"`
}

type SystemModels struct {
	ASR        string `json:"asr"`
	Embeddings string `json:"embeddings"`
	LLM        string `json:"llm"`
}

type SystemInfo struct {
	Offline bool         `json:"offline"`
	Models  SystemModels `json:"models"`
}

type Bookmark struct {
	ID           string    `json:"id"`
	LectureID    string    `json:"lecture_id"`
	LectureTitle string    `json:"lecture_title"`
	T            float64   `json:"t"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BookmarkCreate struct {
	LectureID string  `json:"lecture_id"`
	T         float64 `json:"t"`
	Note      string  `json:"note"`
}

type WatchProgress struct {
	LectureID string    `json:"lecture_id"`
	Position  float64   `json:"position"`
	UpdatedAt time.Time `json:"updated_at"`
}
