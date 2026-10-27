package pipeline

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func extractAudio(videoPath string) (string, error) {
	dir := filepath.Dir(videoPath)
	base := filepath.Base(videoPath)
	audioPath := filepath.Join(dir, "..", "audio", base+".wav")
	if err := os.MkdirAll(filepath.Dir(audioPath), 0o755); err != nil {
		return "", err
	}
	cmd := exec.Command("ffmpeg", "-y", "-i", videoPath, "-vn", "-ac", "1", "-ar", "16000", audioPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), "does not contain any stream") {
			return "", fmt.Errorf("в видео нет звуковой дорожки")
		}
		return "", fmt.Errorf("ffmpeg: %w: %s", err, lastLine(string(out)))
	}
	return audioPath, nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
