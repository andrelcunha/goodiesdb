package rdb

import (
	"encoding/gob"
	"os"

	"github.com/andrelcunha/goodiesdb/internal/core/store"
)

// SaveSnapshot saves the current state of the store to a file
func SaveSnapshot(s *store.Store, filename string) error {
	data := s.GetSnapshot()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)

	// Create a struct to hold both data and expires for encoding
	snapshot := struct {
		Data []map[string]*store.Value
	}{
		Data: data,
	}

	return encoder.Encode(snapshot)
}

// LoadSnapshot loads the state of the store from a file
func LoadSnapshot(s *store.Store, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)

	// Create a struct to decode into
	var snapshot struct {
		Data []map[string]*store.Value
	}

	err = decoder.Decode(&snapshot)
	if err != nil {
		return err
	}

	s.RestoreFromSnapshot(snapshot.Data)
	return nil
}
