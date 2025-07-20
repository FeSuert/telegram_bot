package state

import (
	"sync"
	"testing"
)

func TestStore_GetSet(t *testing.T) {
	s := New()

	if got := s.Get(); got != Disarmed {
		t.Fatalf("default should be DISARMED, got %s", got)
	}

	s.Set(Armed)
	if got := s.Get(); got != Armed {
		t.Fatalf("want ARMED, got %s", got)
	}
}

// Extra assurance that the RWMutex really protects us.
func TestStore_RaceSafety(t *testing.T) {
	const n = 100
	s := New()

	var wg sync.WaitGroup
	wg.Add(n * 2)

	for i := 0; i < n; i++ {
		go func() { s.Set(Armed); wg.Done() }()
		go func() { _ = s.Get(); wg.Done() }()
	}
	wg.Wait()
}
