package store

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andrelcunha/goodiesdb/internal/utils/slice"
)

type Store struct {
	data    []map[string]interface{}
	expires []map[string]time.Time
	mu      sync.RWMutex
	aofChan chan string
}

// NewStore creates a new store
func NewStore(aofChan chan string) *Store {
	data := make([]map[string]interface{}, 16)
	expires := make([]map[string]time.Time, 16)
	for i := range data {
		data[i] = make(map[string]interface{})
		expires[i] = make(map[string]time.Time)
	}
	return &Store{
		data:    data,
		expires: expires,
		aofChan: aofChan,
	}
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// GetSnapshot returns a snapshot of store data for persistence
// This is safe to call as it returns a copy
func (s *Store) GetSnapshot() ([]map[string]interface{}, []map[string]time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create deep copies to avoid data races
	dataCopy := make([]map[string]interface{}, len(s.data))
	expiresCopy := make([]map[string]time.Time, len(s.expires))

	for i := range s.data {
		dataCopy[i] = make(map[string]interface{})
		expiresCopy[i] = make(map[string]time.Time)

		for k, v := range s.data[i] {
			dataCopy[i][k] = v
		}
		for k, v := range s.expires[i] {
			expiresCopy[i][k] = v
		}
	}

	return dataCopy, expiresCopy
}

// RestoreFromSnapshot restores store data from persistence
func (s *Store) RestoreFromSnapshot(data []map[string]interface{}, expires []map[string]time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = data
	s.expires = expires
}

// Test helper methods - only use in tests
// GetListLength returns the length of a list for testing
func (s *Store) GetListLength(dbIndex int, key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if list, ok := s.data[dbIndex][key].([]string); ok {
		return len(list)
	}
	return 0
}

// GetList returns a copy of the list for testing
func (s *Store) GetList(dbIndex int, key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if list, ok := s.data[dbIndex][key].([]string); ok {
		// Return a copy to avoid data races
		result := make([]string, len(list))
		copy(result, list)
		return result
	}
	return nil
}

// SetRawValue sets a raw value for testing (bypasses type safety)
func (s *Store) SetRawValue(dbIndex int, key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[dbIndex][key] = value
}

func (s *Store) AOFChannel() chan string {
	return s.aofChan
}

// Set sets the value for a key
func (s *Store) Set(dbIndex int, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[dbIndex][key] = value
	s.aofChan <- fmt.Sprintf("SET %d %s %s", dbIndex, key, value)
}

// Get gets the value for a key
func (s *Store) Get(dbIndex int, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.isExpired(dbIndex, key) {
		return "", false
	}
	value, ok := s.data[dbIndex][key].(string)
	return value, ok
}

func (s *Store) Del(dbIndex int, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.delKey(dbIndex, key)
	s.aofChan <- fmt.Sprintf("DEL %d %s", dbIndex, key)
}

// Exists checks if a key exists
func (s *Store) Exists(dbIndex int, key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.isExpired(dbIndex, key) {
		return false
	}
	_, ok := s.data[dbIndex][key]
	return ok
}

// SetNx sets the value for a key if the key does not exist
func (s *Store) SetNX(dbIndex int, key, value string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[dbIndex][key]; exists {
		return false
	}
	s.data[dbIndex][key] = value
	s.aofChan <- fmt.Sprintf("SET %d %s %s", dbIndex, key, value)
	return true
}

// Expire sets the expiration time for a key
func (s *Store) Expire(dbIndex int, key string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[dbIndex][key]; exists {
		s.expires[dbIndex][key] = time.Now().Add(ttl)
		s.aofChan <- fmt.Sprintf("EXPIRE %d %s %d", dbIndex, key, int(ttl.Seconds()))
		return true
	}
	return false
}

// Incr increments the value for a key
func (s *Store) Incr(dbIndex int, key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.data[dbIndex][key].(string)
	if !ok {
		value = "0"
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("value is not an integer or out of range")
	}

	intValue++
	s.data[dbIndex][key] = strconv.Itoa(intValue)
	s.aofChan <- fmt.Sprintf("INCR %d %s", dbIndex, key)
	return intValue, nil
}

// Decr decrements the value for a key
func (s *Store) Decr(dbIndex int, key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.data[dbIndex][key].(string)
	if !ok {
		value = "0"
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("value is not an integer or out of range")
	}

	intValue--
	s.data[dbIndex][key] = strconv.Itoa(intValue)
	s.aofChan <- fmt.Sprintf("DECR %d %s", dbIndex, key)
	return intValue, nil
}

// TTL Retrieve the remaining time to live for a key
func (s *Store) TTL(dbIndex int, key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[dbIndex][key]; !ok {
		return -2, nil
	}

	if _, ok := s.expires[dbIndex][key]; !ok {
		return -1, nil
	}

	ttl := time.Until(s.expires[dbIndex][key])
	return int(ttl.Seconds()), nil
}

// LPush inserts values at the begining of a list
func (s *Store) LPush(dbIndex int, key string, values ...string) int {
	logString := fmt.Sprintf("LPUSH %d %s %s", dbIndex, key, strings.Join(values, " "))
	s.mu.Lock()
	defer s.mu.Unlock()

	list, _ := s.data[dbIndex][key].([]string)
	// Reverse values
	slice.Reverse(values)
	list = append(values, list...)

	s.data[dbIndex][key] = list
	s.aofChan <- logString
	return len(list)
}

// RPush inserts values at the end of a list
func (s *Store) RPush(dbIndex int, key string, values ...string) int {
	logString := fmt.Sprintf("RPUSH %d %s %s", dbIndex, key, strings.Join(values, " "))
	s.mu.Lock()
	defer s.mu.Unlock()

	list, _ := s.data[dbIndex][key].([]string)
	list = append(list, values...)
	s.data[dbIndex][key] = list
	s.aofChan <- logString
	return len(list)
}

// LPop removes and returns the first N elements of the list, where N is equal to count, or nil if the list is empty.
func (s *Store) LPop(dbIndex int, key string, pcount *int) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the key has expired
	if s.isExpired(dbIndex, key) {
		return nil, nil
	}

	count := 1
	//if not nil, get the count from the caller
	if pcount != nil {
		count = *pcount
	}

	// Check if count is smaller than 0 and value came from caller
	if count < 0 {
		return nil, fmt.Errorf("value is out of range, must be positive")
	}

	list, ok := s.data[dbIndex][key].([]string)
	if !ok {
		return nil, nil
	}
	len := len(list)
	if len == 0 {
		return nil, nil
	}
	if count > len {
		count = len
	}
	popped := list[:count]

	// Remove the popped elements from the list
	s.data[dbIndex][key] = list[count:]

	// Log the operation
	s.aofChan <- fmt.Sprintf("LPOP %d %s %d", dbIndex, key, count)

	if count == 1 && pcount == nil {
		return popped[0], nil
	} else {
		return popped, nil
	}

}

// RPop removes and returns the last N elements of the list, where N is equal to count, or nil if the list is empty.
func (s *Store) RPop(dbIndex int, key string, pcount *int) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the key has expired
	if s.isExpired(dbIndex, key) {
		return nil, nil
	}

	count := 1
	//if not nil, get the count from the caller
	if pcount != nil {
		count = *pcount
	}

	// Check if count is smaller than 0 and value came from caller
	if count < 0 && pcount != nil {
		return nil, fmt.Errorf("value is out of range, must be positive")
	} else {
		list, ok := s.data[dbIndex][key].([]string)
		if !ok {
			return nil, nil
		}
		len := len(list)
		if len == 0 {
			return nil, nil
		}
		if count > len {
			count = len
		}
		popped := list[(len - count):]

		// Remove the popped elements from the list
		s.data[dbIndex][key] = list[:(len - count)]

		// Log the operation
		s.aofChan <- fmt.Sprintf("RPOP %d %s %d", dbIndex, key, count)

		if count == 1 && pcount == nil {
			return popped[0], nil
		} else {
			return popped, nil
		}
	}
}

// LRange returns the specified elements of the list
func (s *Store) LRange(dbIndex int, key string, start, stop int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the key has expired
	if s.isExpired(dbIndex, key) {
		return nil, nil
	}

	list, ok := s.data[dbIndex][key].([]string)
	if !ok {
		return nil, nil
	}

	len := len(list)

	// Adjust start and stop indices if they are out of bounds
	if start < 0 {
		start = len + start
	}
	if stop < 0 {
		stop = len + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= len {
		stop = len - 1
	}

	if start > stop || start >= len || stop < 0 {
		return []string{}, nil
	}

	return list[start : stop+1], nil
}

// LTrim removes elements from a list
func (s *Store) LTrim(dbIndex int, key string, start, stop int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the key has expired
	if s.isExpired(dbIndex, key) {
		return nil
	}

	list, ok := s.data[dbIndex][key].([]string)
	if !ok {
		return nil
	}

	len := len(list)

	// Adjust start and stop indices if they are out of bounds
	if start < 0 {
		start = len + start
	}
	if stop < 0 {
		stop = len + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= len {
		stop = len - 1
	}

	if start > stop || start >= len {
		s.Del(dbIndex, key)
		return nil
	}

	// Remove the elements from the list
	s.data[dbIndex][key] = list[start : stop+1]

	// Log the operation
	s.aofChan <- fmt.Sprintf("LTRIM %d %s %d %d", dbIndex, key, start, stop)

	return nil
}

// Rename Renames a key and overwrites the destination
func (s *Store) Rename(dbIndex int, key, newkey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the key has expired
	if s.isExpired(dbIndex, key) {
		return nil
	}

	// Check if the new key already exists
	if _, ok := s.data[dbIndex][newkey]; ok {
		// Overwrite the destination
		s.delKey(dbIndex, newkey)
	}
	s.data[dbIndex][newkey] = s.data[dbIndex][key]
	s.delKey(dbIndex, key)

	// Log the operation
	s.aofChan <- fmt.Sprintf("RENAME %d %s %s", dbIndex, key, newkey)

	return nil
}

// Tupe determine the type of value stored at a key
func (s *Store) Type(dbIndex int, key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// verify if key exists
	if val, exists := s.data[dbIndex][key]; exists {
		switch val.(type) {
		case string:
			return "string"
		case []string:
			return "list"
			//add more types below
		}
	}
	return "none"
}

// Keys returns all keys matching a pattern
func (s *Store) Keys(dbIndex int, pattern string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := []string{}
	// Convert Redis-like pattern to a valid regex
	regexPattern := "^" + strings.ReplaceAll(pattern, "*", ".*") + "$"
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, err
	}

	for key := range s.data[dbIndex] {
		if re.MatchString(key) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *Store) FlushDb(dbIndex int) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flushDb(dbIndex)
	s.aofChan <- fmt.Sprintf("FLUSHDB %d", dbIndex)
	return "OK"
}

func (s *Store) FlushAll() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	for dbIndex := range s.data {
		s.flushDb(dbIndex)
	}
	s.aofChan <- "FLUSHALL"
	return "OK"
}
