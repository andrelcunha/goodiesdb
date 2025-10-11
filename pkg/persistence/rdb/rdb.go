package rdb

import (
	"encoding/gob"
	"os"

	"com.github.andrelcunha.goodiesdb/pkg/store"
)

// SaveSnapshot saves the current state of the store to a file
func SaveSnapshot(s *store.Store, filename string) error {
	s.RLock()
	defer s.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(s)
}

// LoadSnapshot loads the state of the store from a file
func LoadSnapshot(s *store.Store, filename string) error {
	s.Lock()
	defer s.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	return decoder.Decode(s)
}
