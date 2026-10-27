package pipeline

import "aba/internal/mlclient"

type Chunk struct {
	Start float64
	End   float64
	Text  string
}

func chunkSegments(segs []mlclient.Segment, minSec, maxSec float64) []Chunk {
	if len(segs) == 0 {
		return nil
	}
	var out []Chunk
	var cur Chunk
	for _, s := range segs {
		if cur.Text == "" {
			cur.Start = s.Start
		}
		cur.Text += " " + s.Text
		cur.End = s.End
		if cur.End-cur.Start >= minSec || (cur.End-cur.Start <= maxSec && s.End-cur.Start >= maxSec) {
			out = append(out, cur)
			cur = Chunk{}
		}
	}
	if cur.Text != "" {
		out = append(out, cur)
	}
	return out
}
