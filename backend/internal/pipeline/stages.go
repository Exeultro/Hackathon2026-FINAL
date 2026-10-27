package pipeline

import (
	"context"
	"fmt"
	"log"
	"os"

	"aba/internal/mlclient"
	"aba/internal/store"
)

func (p *Pipeline) process(ctx context.Context, lectureID string) error {
	lec, err := p.store.GetLecture(ctx, lectureID)
	if err != nil {
		return err
	}

	var videoPath string
	if lec.VideoPath != nil && *lec.VideoPath != "" {
		videoPath = *lec.VideoPath
	} else if lec.SourceURL != nil && *lec.SourceURL != "" {
		_ = p.store.SetLectureStatus(ctx, lectureID, store.StatusDownloading, 0.05, nil)
		var title string
		videoPath, title, err = downloadVideo(*lec.SourceURL, lectureID, p.dataDir)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}
		_ = p.store.SetLectureVideo(ctx, lectureID, videoPath)
		if title != "" && lec.Title == *lec.SourceURL {
			_, _ = p.store.UpdateLecture(ctx, lectureID, &title, false, nil)
		}
	} else {
		return fmt.Errorf("no video source")
	}

	// extracting_audio
	_ = p.store.SetLectureStatus(ctx, lectureID, store.StatusExtractingAudio, 0.15, nil)
	audioPath, err := extractAudio(videoPath)
	if err != nil {
		return fmt.Errorf("extract audio: %w", err)
	}
	defer os.Remove(audioPath)
	duration, _ := probeDuration(videoPath)
	if duration > 0 {
		_ = p.store.SetDuration(ctx, lectureID, duration)
	}
	if lec.ThumbnailPath == nil {
		if thumb, err := extractThumbnail(videoPath, lectureID, p.dataDir, duration); err != nil {
			log.Printf("thumbnail %s: %v", lectureID, err)
		} else {
			_ = p.store.SetLectureThumbnail(ctx, lectureID, thumb)
		}
	}

	// transcribing
	_ = p.store.SetLectureStatus(ctx, lectureID, store.StatusTranscribing, 0.25, nil)
	lang := ""
	if lec.Language != nil {
		lang = *lec.Language
	}
	asr, err := p.ml.ASR(ctx, audioPath, lang)
	if err != nil {
		return fmt.Errorf("asr: %w", err)
	}
	segs := make([]store.Segment, len(asr.Segments))
	for i, s := range asr.Segments {
		segs[i] = store.Segment{Idx: i, Start: s.Start, End: s.End, Text: s.Text}
	}
	if err := p.store.SaveSegments(ctx, lectureID, segs); err != nil {
		return fmt.Errorf("save segments: %w", err)
	}
	if duration <= 0 {
		duration = asr.Duration
	}
	_ = p.store.SetDurationAndLanguage(ctx, lectureID, duration, asr.Language)

	// indexing
	_ = p.store.SetLectureStatus(ctx, lectureID, store.StatusIndexing, 0.55, nil)
	chunks := chunkSegments(asr.Segments, 20, 40)
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = "passage: " + c.Text
	}
	vectors, err := p.ml.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	storeChunks := make([]store.Chunk, len(chunks))
	for i, c := range chunks {
		storeChunks[i] = store.Chunk{Start: c.Start, End: c.End, Text: c.Text}
	}
	if err := p.store.SaveChunks(ctx, lectureID, storeChunks, vectors); err != nil {
		return fmt.Errorf("save chunks: %w", err)
	}

	if err := p.store.SetLectureStatus(ctx, lectureID, store.StatusReady, 1.0, nil); err != nil {
		return err
	}
	go p.generateSummary(context.Background(), lectureID, asr.Segments)
	return nil
}

func (p *Pipeline) RetrySummary(lectureID string) error {
	ctx := context.Background()
	segs, err := p.store.ListSegments(ctx, lectureID)
	if err != nil {
		return err
	}
	ms := make([]mlclient.Segment, len(segs))
	for i, s := range segs {
		ms[i] = mlclient.Segment{Start: s.Start, End: s.End, Text: s.Text}
	}
	if err := p.store.SetSummaryStatus(ctx, lectureID, store.SummaryPending, nil); err != nil {
		return err
	}
	go p.generateSummary(ctx, lectureID, ms)
	return nil
}

func (p *Pipeline) generateSummary(ctx context.Context, lectureID string, segs []mlclient.Segment) {
	_ = p.store.SetSummaryStatus(ctx, lectureID, store.SummaryProcessing, nil)
	resp, err := p.ml.Summary(ctx, segs)
	if err != nil {
		log.Printf("summary %s: %v", lectureID, err)
		msg := err.Error()
		_ = p.store.SetSummaryStatus(ctx, lectureID, store.SummaryFailed, &msg)
		return
	}
	if err := p.store.SaveSummary(ctx, lectureID, resp.Summary, resp.Outline, resp.KeyTerms); err != nil {
		msg := err.Error()
		_ = p.store.SetSummaryStatus(ctx, lectureID, store.SummaryFailed, &msg)
		return
	}
	_ = p.store.SetSummaryStatus(ctx, lectureID, store.SummaryReady, nil)
}
