package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrEmailTaken = errors.New("email taken")
)

type LectureStatus string
type SummaryStatus string

const (
	StatusQueued          LectureStatus = "queued"
	StatusDownloading     LectureStatus = "downloading"
	StatusExtractingAudio LectureStatus = "extracting_audio"
	StatusTranscribing    LectureStatus = "transcribing"
	StatusIndexing        LectureStatus = "indexing"
	StatusReady           LectureStatus = "ready"
	StatusFailed          LectureStatus = "failed"
)

const (
	SummaryPending    SummaryStatus = "pending"
	SummaryProcessing SummaryStatus = "processing"
	SummaryReady      SummaryStatus = "ready"
	SummaryFailed     SummaryStatus = "failed"
)

type Lecture struct {
	ID            string
	Title         string
	CourseID      *string
	Status        LectureStatus
	Progress      float64
	Error         *string
	SummaryStatus SummaryStatus
	Source        string
	SourceURL     *string
	PlaybackType  *string
	PlaybackURL   *string
	ProviderID    *string
	Duration      *float64
	Language      *string
	ThumbnailPath *string
	VideoPath     *string
	AudioPath     *string
	Owner         *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Course struct {
	ID           string
	Title        string
	LectureCount int
	Owner        *string
	CreatedAt    time.Time
}

type User struct {
	ID               string
	Email            string
	Name             string
	CreatedAt        time.Time
	TelegramID       *int64
	TelegramUsername *string
}

type Store interface {
	ListLectures(ctx context.Context, courseID *string, status *string, viewer string) ([]Lecture, error)
	GetLecture(ctx context.Context, id string) (*Lecture, error)
	CreateLecture(ctx context.Context, l *Lecture) error
	UpdateLecture(ctx context.Context, id string, title *string, setCourse bool, courseID *string) (*Lecture, error)
	DeleteLecture(ctx context.Context, id string) error
	SetLectureStatus(ctx context.Context, id string, status LectureStatus, progress float64, errMsg *string) error
	SetSummaryStatus(ctx context.Context, id string, status SummaryStatus, errMsg *string) error
	SetLectureVideo(ctx context.Context, id, videoPath string) error
	SetLectureThumbnail(ctx context.Context, id, thumbPath string) error

	SetDurationAndLanguage(ctx context.Context, id string, duration float64, language string) error
	SetDuration(ctx context.Context, id string, duration float64) error
	SaveSegments(ctx context.Context, lectureID string, segs []Segment) error
	ListSegments(ctx context.Context, lectureID string) ([]Segment, error)
	SaveChunks(ctx context.Context, lectureID string, chunks []Chunk, vectors [][]float32) error
	SearchChunks(ctx context.Context, vec []float32, limit int, lectureID, courseID *string, viewer string) ([]SearchHit, error)
	SaveSummary(ctx context.Context, lectureID, summary string, outline, keyTerms any) error
	GetSummary(ctx context.Context, lectureID string) (*SummaryData, error)

	ListCourses(ctx context.Context, viewer string) ([]Course, error)
	GetCourse(ctx context.Context, id string) (*Course, error)
	CreateCourse(ctx context.Context, title, owner string) (*Course, error)
	UpdateCourse(ctx context.Context, id, title string) (*Course, error)
	DeleteCourse(ctx context.Context, id string) error

	ListBookmarks(ctx context.Context, clientID string, lectureID *string) ([]Bookmark, error)
	CreateBookmark(ctx context.Context, b *Bookmark) (*Bookmark, error)
	UpdateBookmark(ctx context.Context, clientID, id string, note *string, t *float64) (*Bookmark, error)
	DeleteBookmark(ctx context.Context, clientID, id string) error

	ListProgress(ctx context.Context, clientID string) ([]WatchProgress, error)
	SaveProgress(ctx context.Context, clientID, lectureID string, position float64) (*WatchProgress, error)
	ClearProgress(ctx context.Context, clientID string) error

	CreateUser(ctx context.Context, id, email, name, passwordHash string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, string, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ConsumeRefreshToken(ctx context.Context, tokenHash string) (string, error)
	DeleteRefreshToken(ctx context.Context, tokenHash string) error

	SaveTelegramCode(ctx context.Context, code, userID string, expiresAt time.Time) error
	LinkTelegram(ctx context.Context, code string, telegramID int64, username, anonOwner string) (*User, error)
	GetUserByTelegram(ctx context.Context, telegramID int64) (*User, error)
	UnlinkTelegram(ctx context.Context, userID string) error
}

func NewID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func Visible(owner *string, viewer string) bool {
	return owner == nil || (viewer != "" && *owner == viewer)
}
