package main

import "unicode"

// Match is the result of fuzzy-matching a pattern against a candidate string.
type Match struct {
	Score     float64
	Positions []int // indexes into the candidate rune slice that were matched
}

// isBoundary reports whether the rune at position i in runes starts a new
// "word" — i.e. it follows a separator or is an upper-case letter following a
// lower-case one (camelCase). Boundary matches score higher because they line
// up with how people mentally chunk a name (kube_config -> "kube" | "config").
func isBoundary(runes []rune, i int) bool {
	if i == 0 {
		return true
	}
	prev := runes[i-1]
	switch prev {
	case '_', '-', '/', '.', ' ':
		return true
	}
	return unicode.IsLower(prev) && unicode.IsUpper(runes[i])
}

// FuzzyMatch performs a case-insensitive subsequence match of pattern against
// text. It returns the matched positions (for highlighting) and a score that
// rewards contiguous runs and word-boundary hits, and lightly prefers shorter
// candidates. Greedy earliest-match is good enough for short repo names.
func FuzzyMatch(pattern, text string) (Match, bool) {
	if pattern == "" {
		// Everything matches an empty query; no positions, neutral score.
		return Match{}, true
	}

	tr := []rune(text)
	pr := []rune(pattern)

	positions := make([]int, 0, len(pr))
	ti := 0
	for _, pc := range pr {
		lpc := unicode.ToLower(pc)
		found := false
		for ; ti < len(tr); ti++ {
			if unicode.ToLower(tr[ti]) == lpc {
				positions = append(positions, ti)
				ti++
				found = true
				break
			}
		}
		if !found {
			return Match{}, false
		}
	}

	score := 0.0
	prev := -2
	for _, p := range positions {
		score += 1 // base credit per matched rune
		if p == prev+1 {
			score += 3 // contiguous run
		}
		if isBoundary(tr, p) {
			score += 2 // start of a word
		}
		prev = p
	}
	// Slight preference for shorter candidates and earlier first match.
	score -= 0.01 * float64(len(tr))
	if len(positions) > 0 {
		score -= 0.05 * float64(positions[0])
	}

	return Match{Score: score, Positions: positions}, true
}
