package store

// delKey deletes a key from the store and its expiration
func (s *Store) delKey(dbIndex int, key string) {
	delete(s.data[dbIndex], key)
}

// flushDb flushes the database
func (s *Store) flushDb(dbIndex int) {
	s.data[dbIndex] = make(map[string]*Value)
}
