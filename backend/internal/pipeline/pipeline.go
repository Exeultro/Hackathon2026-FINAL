package pipeline

import (
	"context"
	"log"

	"aba/internal/mlclient"
	"aba/internal/store"
)

type Pipeline struct {
	jobs    chan string
	store   store.Store
	ml      *mlclient.Client
	dataDir string
}

func New(s store.Store, ml *mlclient.Client, dataDir string) *Pipeline {
	return &Pipeline{jobs: make(chan string, 64), store: s, ml: ml, dataDir: dataDir}
}

func (p *Pipeline) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		go p.worker(ctx, i)
	}
}

func (p *Pipeline) Enqueue(lectureID string) {
	p.jobs <- lectureID
}

func (p *Pipeline) worker(ctx context.Context, id int) {
	for lectureID := range p.jobs {
		if err := p.process(ctx, lectureID); err != nil {
			log.Printf("pipeline[%d] %s failed: %v", id, lectureID, err)
			msg := err.Error()
			if e := p.store.SetLectureStatus(ctx, lectureID, store.StatusFailed, 0, &msg); e != nil {
				log.Printf("pipeline[%d] %s: cannot set failed status: %v", id, lectureID, e)
			}
		}
	}
}
