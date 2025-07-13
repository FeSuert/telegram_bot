package state

import "sync"

type AlarmState string

const (
	Armed AlarmState = "ARMED"
	Disarmed  = "DISARMED"
)

type Store struct {
	sync.RWMutex
	val AlarmState
}

func New() *Store { return &Store{val: Disarmed} }

func (s *Store) Get() AlarmState {
	s.RLock()
	defer s.RUnlock()
	return s.val
}

func (s *Store) Set(v AlarmState) {
	s.Lock()
	s.val = v
	s.Unlock()
}
