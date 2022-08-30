package pikachu

import "sync"

type collection struct {
	Name string
	Data map[string]any
	sync.RWMutex
}

func (s *collection) Exist(key string) bool {
	if key == "" {
		return false
	}
	s.RLock()
	defer s.RUnlock()
	_, ok := s.Data[key]
	return ok
}

func (s *collection) Load(key string) any {
	s.RLock()
	defer s.RUnlock()
	return s.Data[key]
}

func (s *collection) Loads(keys []string) []any {
	res := make([]any, len(keys))
	s.RLock()
	for i := range keys {
		res[i] = s.Data[keys[i]]
	}
	s.RUnlock()
	return res
}

func (s *collection) Store(key string, value any) {
	s.Lock()
	defer s.Unlock()
	s.Data[key] = value
}

func (s *collection) Range(f func(key string, value any)) {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.Data {
		f(k, v)
	}
}
