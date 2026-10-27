package storage

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

const MaxUploadBytes = 2 << 30

var ErrUnsupported = errors.New("unsupported format")

type Files struct {
	root string
}

func New(root string) (*Files, error) {
	if err := os.MkdirAll(filepath.Join(root, "videos"), 0o755); err != nil {
		return nil, err
	}
	return &Files{root: root}, nil
}

func (f *Files) SaveVideo(lectureID string, header *multipart.FileHeader, src multipart.File) (string, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".mp4" && ext != ".webm" {
		return "", ErrUnsupported
	}
	dstPath := filepath.Join(f.root, "videos", lectureID+ext)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("create: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("copy: %w", err)
	}
	return dstPath, nil
}

func (f *Files) Open(path string) (*os.File, error) {
	return os.Open(path)
}

func (f *Files) Delete(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}

func (f *Files) OpenThumb(path string) (*os.File, error) {
	return os.Open(path)
}
