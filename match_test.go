package main

import "testing"

func TestFuzzyMatchSubsequence(t *testing.T) {
	cases := []struct {
		pattern, text string
		want          bool
		wantPos       []int
	}{
		{"kc", "kube_config", true, []int{0, 5}},     // boundary hits: k…|config
		{"kbcfg", "kube_config", true, []int{0, 2, 5, 8, 10}},
		{"config", "kube_config", true, []int{5, 6, 7, 8, 9, 10}},
		{"xyz", "kube_config", false, nil},
		{"", "anything", true, nil}, // empty query matches with no positions
	}
	for _, c := range cases {
		m, ok := FuzzyMatch(c.pattern, c.text)
		if ok != c.want {
			t.Errorf("FuzzyMatch(%q,%q) ok=%v want %v", c.pattern, c.text, ok, c.want)
			continue
		}
		if !equalInts(m.Positions, c.wantPos) {
			t.Errorf("FuzzyMatch(%q,%q) pos=%v want %v", c.pattern, c.text, m.Positions, c.wantPos)
		}
	}
}

func TestContiguousBeatsScattered(t *testing.T) {
	// "config" as a contiguous run should outscore the same letters scattered.
	tight, _ := FuzzyMatch("config", "kube_config")
	loose, _ := FuzzyMatch("cfg", "kube_config")
	if tight.Score <= loose.Score {
		t.Errorf("contiguous score %.2f should beat scattered %.2f", tight.Score, loose.Score)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
