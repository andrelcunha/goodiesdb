package store

import "time"

// isExpired checks if a key has expired
func (s *Store) isExpired(dbIndex int, key string) bool {
	if exp, exists := s.Expires[dbIndex][key]; exists {
		if time.Now().After(exp) {
			s.delKey(dbIndex, key)
			return true
		}
	}
	return false
}

// delKey deletes a key from the store and its expiration
func (s *Store) delKey(dbIndex int, key string) {
	delete(s.Data[dbIndex], key)
	delete(s.Expires[dbIndex], key)
}

// flushDb flushes the database
func (s *Store) flushDb(dbIndex int) {
	s.Data[dbIndex] = make(map[string]interface{})
	s.Expires[dbIndex] = make(map[string]time.Time)
}
