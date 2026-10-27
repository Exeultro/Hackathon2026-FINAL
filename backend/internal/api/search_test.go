package api

import (
	"reflect"
	"testing"
)

func TestHighlights(t *testing.T) {
	cases := []struct {
		text, query string
		want        [][]int
	}{
		{"Квадрат гипотенузы равен сумме квадратов катетов.", "гипотенуза", [][]int{{8, 18}}},
		{"Квадрат гипотенузы равен сумме квадратов катетов.", "квадрат", [][]int{{0, 7}, {31, 40}}},
		{"emoji 😀 Пифагор", "пифагора", [][]int{{9, 16}}},
		{"ничего общего", "теорема", [][]int{}},
		{"а и в", "и", [][]int{}},
	}
	for _, c := range cases {
		if got := highlights(c.text, c.query); !reflect.DeepEqual(got, c.want) {
			t.Errorf("highlights(%q, %q) = %v, want %v", c.text, c.query, got, c.want)
		}
	}
}

func TestSourceOf(t *testing.T) {
	for host, want := range map[string]string{
		"www.youtube.com": "youtube", "youtu.be": "youtube", "m.youtube.com": "youtube",
		"vimeo.com": "vimeo", "player.vimeo.com": "vimeo", "example.com": "url",
	} {
		if got := sourceOf(host); got != want {
			t.Errorf("sourceOf(%q) = %q, want %q", host, got, want)
		}
	}
}
