package pipeline

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func extractThumbnail(videoPath, lectureID, dataDir string, duration float64) (string, error) {
	out := filepath.Join(dataDir, "thumbs", lectureID+".jpg")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	at := 5.0
	if duration > 0 && duration < 10 {
		at = duration / 2
	}
	cmd := exec.Command("ffmpeg",
		"-y",
		"-ss", fmt.Sprintf("%.2f", at),
		"-i", videoPath,
		"-frames:v", "1",
		"-vf", "scale=480:-2",
		out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg thumb: %w: %s", err, string(b))
	}
	return out, nil
}
