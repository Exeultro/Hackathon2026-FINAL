package pipeline

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func downloadVideo(sourceURL, lectureID, dataDir string) (string, string, error) {
	out := filepath.Join(dataDir, "videos", lectureID+".mp4")
	cmd := exec.Command("yt-dlp",
		"-f", "bv*+ba/b",
		"-S", "res:720,vcodec:h264,acodec:m4a",
		"--merge-output-format", "mp4",
		"--no-playlist",
		"--js-runtimes", "node",
		"--no-simulate", "--print", "after_move:title",
		"-o", out,
		sourceURL,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("yt-dlp: %w: %s", err, lastLine(stderr.String()))
	}
	return out, strings.TrimSpace(stdout.String()), nil
}
