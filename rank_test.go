package main

import "testing"

// A repo that has been opened before should rank above an equally-good textual
// match that hasn't — this is the whole point of the frecency signal.
func TestFrecencyReordersEqualMatches(t *testing.T) {
	// Symmetric names so the textual match score is identical — the only
	// differentiator is frecency.
	repos := []string{"service_alpha", "service_bravo"}
	fre := &Frecency{entries: map[string]entry{}}
	const now int64 = 1_000_000

	// Without any usage, ties break alphabetically: alpha first.
	m := newModel(repos, fre, now, 2.0)
	m.ti.SetValue("service")
	m.recompute()
	if m.results[0].name != "service_alpha" {
		t.Fatalf("cold start: want service_alpha first, got %s", m.results[0].name)
	}

	// Open bravo a few times "just now"; it should now win.
	fre.entries["service_bravo"] = entry{Count: 3, Last: now}
	m.recompute()
	if m.results[0].name != "service_bravo" {
		t.Fatalf("after usage: want service_bravo first, got %s", m.results[0].name)
	}
}

func TestEmptyQueryRanksByFrecency(t *testing.T) {
	repos := []string{"aaa", "bbb", "ccc"}
	fre := &Frecency{entries: map[string]entry{"ccc": {Count: 5, Last: 1_000_000}}}
	m := newModel(repos, fre, 1_000_000, 2.0)
	// Empty query -> most-used repo surfaces first.
	if m.results[0].name != "ccc" {
		t.Fatalf("empty query: want ccc first, got %s", m.results[0].name)
	}
}
