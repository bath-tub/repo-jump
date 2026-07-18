package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// entry records how often and how recently a repo was chosen.
type entry struct {
	Count int   `json:"count"`
	Last  int64 `json:"last"` // unix seconds of the most recent open
}

// Frecency maps repo name -> usage entry. It is the tool's only piece of
// persisted state and the sole relevance signal — portable to any user,
// since it depends on nothing about the local machine or git history.
type Frecency struct {
	path    string
	entries map[string]entry
}

// LoadFrecency reads the frecency store, returning an empty (usable) store if
// the file does not exist yet.
func LoadFrecency() (*Frecency, error) {
	path := filepath.Join(dataDir(), "frecency.json")
	f := &Frecency{path: path, entries: map[string]entry{}}

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return f, nil
		}
		return nil, err
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &f.entries); err != nil {
			// Corrupt store shouldn't brick the tool — start fresh.
			f.entries = map[string]entry{}
		}
	}
	return f, nil
}

// Score returns the frecency weight for name at time now (unix seconds), using
// zoxide-style recency buckets multiplied by visit count. Never opened -> 0.
func (f *Frecency) Score(name string, now int64) float64 {
	e, ok := f.entries[name]
	if !ok {
		return 0
	}
	age := now - e.Last
	var mult float64
	switch {
	case age < 3600: // < 1 hour
		mult = 4
	case age < 86400: // < 1 day
		mult = 2
	case age < 604800: // < 1 week
		mult = 0.5
	default:
		mult = 0.25
	}
	return float64(e.Count) * mult
}

// Bump records an open of name at time now and persists the store.
func (f *Frecency) Bump(name string, now int64) error {
	e := f.entries[name]
	e.Count++
	e.Last = now
	f.entries[name] = e
	return f.save()
}

func (f *Frecency) save() error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(f.entries)
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, b, 0o644)
}
