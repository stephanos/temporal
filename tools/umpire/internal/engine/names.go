package engine

import (
	"slices"
	"sync"
)

// claimNames remembers the Property and Scenario names declared on one machine. Two declarations
// of a kind under one name share a Definition ID, so every case, fingerprint and Contract that
// names one would silently name the other.
type claimNames struct {
	mu    sync.Mutex
	seen  map[string]bool
	twice []string
}

func (n *claimNames) declare(kind, name string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	key := kind + " " + name
	if n.seen == nil {
		n.seen = map[string]bool{}
	}
	if n.seen[key] {
		n.twice = append(n.twice, key)
	}
	n.seen[key] = true
}

// duplicate is the error of one name a kind declares twice, or nil.
func (n *claimNames) duplicate(machine, kind, name string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if key := kind + " " + name; slices.Contains(n.twice, key) {
		return declaredTwice(machine, key)
	}
	return nil
}

func declaredTwice(machine, key string) error {
	return errorf("machine "+machine,
		"%s is declared twice, and both declarations would share one Definition ID; rename one", key)
}
