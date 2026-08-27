package persistence

func (s *Store) Replay() error { s.load(); return nil }
