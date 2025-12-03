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
	data    []map[string]Value
	mu      sync.RWMutex
	aofChan chan string
}

// NewStore creates a new store
func NewStore(aofChan chan string) *Store {
	data := make([]map[string]Value, 16)
	for i := range data {
		data[i] = make(map[string]Value)
	}
	return &Store{
		data:    data,
		aofChan: aofChan,
	}
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

var ErrNoSuchKey = fmt.Errorf("no such key")

// GetSnapshot returns a snapshot of store data for persistence
// This is safe to call as it returns a copy
func (s *Store) GetSnapshot() []map[string]Value {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create deep copies to avoid data races
	dataCopy := make([]map[string]Value, len(s.data))

	for i := range s.data {
		dataCopy[i] = make(map[string]Value)

		for k, v := range s.data[i] {
			dataCopy[i][k] = v
		}

	}

	return dataCopy
}

// RestoreFromSnapshot restores store data from persistence
func (s *Store) RestoreFromSnapshot(data []map[string]Value) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = data
}

// Test helper methods - only use in tests
// GetListLength returns the length of a list for testing
func (s *Store) GetListLength(dbIndex int, key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if value, ok := s.data[dbIndex][key]; ok == true {
		list, err := value.AsList()
		if err != nil {
			return 0
		}
		return len(list)
	}
	return 0
}

// GetList returns a copy of the list for testing
func (s *Store) GetList(dbIndex int, key string) []any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if value, ok := s.data[dbIndex][key]; ok == true {
		list, err := value.AsList()
		if err != nil {
			return nil
		}
		// Return a copy to avoid data races
		result := make([]any, len(list))
		copy(result, list)
		return result
	}
	return nil
}

// SetRawValue sets a raw value for testing (bypasses type safety)
func (s *Store) SetRawValue(dbIndex int, key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := Value{Data: value}
	s.data[dbIndex][key] = data
}

func (s *Store) AOFChannel() chan string {
	return s.aofChan
}

// Set sets the value for a key
// Consider ret
func (s *Store) Set(dbIndex int, key string, rawValue any) bool {

	s.mu.Lock()
	defer s.mu.Unlock()
	// write to AOF before setting the value (WAL)
	s.aofChan <- fmt.Sprintf("SET %d %s %v", dbIndex, key, rawValue)
	var value *Value
	switch v := rawValue.(type) {
	case string:
		value = NewStringValue(v)
	case []any:
		value = NewListValue(v)
	case map[string]any:
		value = NewHashValue(v)
	case map[string]struct{}:
		value = NewSetValue(v)
	case map[string]float64:
		value = NewZSetValue(v)
	default:
		// Fallback to string representation
		value = NewStringValue(fmt.Sprintf("%v", rawValue))
	}
	s.data[dbIndex][key] = *value
	return true
}

// Get gets the value for a key
func (s *Store) Get(dbIndex int, key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[dbIndex][key]
	if value.IsExpired() {
		return nil, false
	}
	if !ok {
		return nil, false
	}
	return value, ok
}

// GetRange gets a substring of the string value for a key
func (s *Store) GetRange(dbIndex int, key string, start, end int) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[dbIndex][key]
	if !ok {
		return "", ErrNoSuchKey
	}
	if value.IsExpired() {
		return "", ErrNoSuchKey
	}
	strValue, ok := value.Data.(string)
	if !ok {
		return "", fmt.Errorf("value is not a string")
	}
	if start < 0 {
		start = len(strValue) + start
	}
	if end < 0 {
		end = len(strValue) + end
	}
	if start < 0 {
		start = 0
	}
	if end >= len(strValue) {
		end = len(strValue) - 1
	}
	if start > end {
		return "", nil
	}
	return strValue[start : end+1], nil
}

func (s *Store) Del(dbIndex int, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.delKey(dbIndex, key)
	s.aofChan <- fmt.Sprintf("DEL %d %s", dbIndex, key)
}

// Exists checks if a key exists
func (s *Store) Exists(dbIndex int, keys ...string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, key := range keys {
		value, ok := s.data[dbIndex][key]
		if ok && !value.IsExpired() && value.Data != nil {
			count++
		}
	}

	return count
}

// StrLen returns the length of the string value for a key
func (s *Store) StrLen(dbIndex int, key string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[dbIndex][key]
	if !ok {
		return 0, ErrNoSuchKey
	}
	if value.IsExpired() {
		return 0, ErrNoSuchKey
	}
	strValue, ok := value.Data.(string)
	if !ok {
		return 0, ErrWrongType
	}
	return len(strValue), nil
}

// SetNx sets the value for a key if the key does not exist
func (s *Store) SetNX(dbIndex int, key, value string) int {
	if s.Exists(dbIndex, key) > 0 {
		return 0
	}
	if s.Set(dbIndex, key, value) {
		return 1
	}
	return 0
}

// Expire sets the expiration time for a key
func (s *Store) Expire(dbIndex int, key string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value, exists := s.data[dbIndex][key]; exists {
		expiration := time.Now().Add(ttl)
		value.ExpiresAt = &expiration
		s.data[dbIndex][key] = value
		s.aofChan <- fmt.Sprintf("EXPIRE %d %s %d", dbIndex, key, int(ttl.Seconds()))
		return true
	}
	return false
}

// Incr increments the value for a key
func (s *Store) Incr(dbIndex int, key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.data[dbIndex][key]
	if !ok {
		value = Value{Data: "0", Type: TypeString}
	}
	if value.Type != TypeString {
		return 0, ErrNotInteger
	}

	intValue, err := strconv.Atoi(value.Data.(string))
	if err != nil {
		return 0, ErrNotInteger
	}
	intValue++
	value.Data = strconv.Itoa(intValue)
	s.data[dbIndex][key] = value
	s.aofChan <- fmt.Sprintf("INCR %d %s", dbIndex, key)
	return intValue, nil
}

// Decr decrements the value for a key
func (s *Store) Decr(dbIndex int, key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.data[dbIndex][key]
	if !ok {
		value = Value{Data: "0", Type: TypeString}
	}
	if value.Type != TypeString {
		return 0, ErrNotInteger
	}

	intValue, err := strconv.Atoi(value.Data.(string))
	if err != nil {
		return 0, ErrNotInteger
	}
	intValue--
	value.Data = strconv.Itoa(intValue)
	s.data[dbIndex][key] = value
	s.aofChan <- fmt.Sprintf("DECR %d %s", dbIndex, key)
	return intValue, nil
}

// TTL Retrieve the remaining time to live for a key
func (s *Store) TTL(dbIndex int, key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.data[dbIndex][key]
	if !ok {
		return -2, nil
	}
	if value.ExpiresAt == nil {
		return -1, nil
	}
	ttl := time.Until(*value.ExpiresAt)
	return int(ttl.Seconds()), nil
}

// LPush inserts values at the begining of a list
func (s *Store) LPush(dbIndex int, key string, values ...any) int {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%v", v)
	}
	s.aofChan <- fmt.Sprintf("LPUSH %d %s %s", dbIndex, key, strings.Join(strValues, " "))
	if len(values) > 1 {
		slice.Reverse(values)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.data[dbIndex][key]
	if !ok {
		s.data[dbIndex][key] = *NewListValue(values)
		return len(values)
	}
	list, _ := value.AsList()
	list = append(values, list...)
	value.Data = list
	s.data[dbIndex][key] = value
	return len(list)
}

// RPush inserts values at the end of a list
func (s *Store) RPush(dbIndex int, key string, values ...any) int {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%v", v)
	}
	s.aofChan <- fmt.Sprintf("RPUSH %d %s %s", dbIndex, key, strings.Join(strValues, " "))
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.data[dbIndex][key]
	if !ok {
		s.data[dbIndex][key] = *NewListValue(values)
		return len(values)
	}
	list, _ := value.AsList()
	list = append(list, values...)
	value.Data = list
	s.data[dbIndex][key] = value
	return len(list)
}

// LPop removes and returns the first N elements of the list, where N is equal to count, or nil if the list is empty.
func (s *Store) LPop(dbIndex int, key string, pcount *int) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.data[dbIndex][key]
	if !ok {
		return nil, nil
	}
	// Check if the key has expired
	if value.IsExpired() {
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

	list, err := value.AsList()
	if err != nil {
		return nil, err
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
	value.Data = list[count:]
	s.data[dbIndex][key] = value

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

	value, ok := s.data[dbIndex][key]
	if !ok {
		return nil, nil
	}

	// Check if the key has expired
	if value.IsExpired() {
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
		list, err := value.AsList()
		if err != nil {
			return nil, err
		}

		len := len(list)
		if len == 0 {
			return nil, nil
		}
		if count > len {
			count = len
		}
		popped := list[(len - count):]
		value.Data = list[:(len - count)]

		// Remove the popped elements from the list
		s.data[dbIndex][key] = value

		// Log the operation
		s.aofChan <- fmt.Sprintf("RPOP %d %s %d", dbIndex, key, count)

		if count == 1 && pcount == nil {
			return popped[0], nil
		} else {
			return popped, nil
		}
	}
}

// LRange returns the elements of a list between start and stop
func (s *Store) LRange(dbIndex int, key string, start, stop int) ([]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.data[dbIndex][key]
	if !ok {
		return nil, nil
	}

	// Check if the key has expired
	if value.IsExpired() {
		return nil, nil
	}
	list, err := value.AsList()
	if err != nil {
		return nil, err
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
		return []any{}, nil
	}

	return list[start : stop+1], nil
}

// LTrim trims a list to the specified range
func (s *Store) LTrim(dbIndex int, key string, start, stop int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.data[dbIndex][key]
	if !ok {
		return nil
	}
	// Check if the key has expired
	if value.IsExpired() {
		return nil
	}

	list, err := value.AsList()
	if err != nil {
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
	value.Data = list[start : stop+1]
	s.data[dbIndex][key] = value

	// Log the operation
	s.aofChan <- fmt.Sprintf("LTRIM %d %s %d %d", dbIndex, key, start, stop)

	return nil
}

// Rename Renames a key and overwrites the destination
func (s *Store) Rename(dbIndex int, oldKey, newKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the key has expired
	value, ok := s.data[dbIndex][oldKey]
	if !ok {
		return ErrNoSuchKey
	}
	if value.IsExpired() {
		return nil
	}

	// Check if the new key already exists
	if _, ok := s.data[dbIndex][newKey]; ok {
		// Overwrite the destination
		s.delKey(dbIndex, newKey)
	}
	s.data[dbIndex][newKey] = value
	s.delKey(dbIndex, oldKey)

	// Log the operation
	s.aofChan <- fmt.Sprintf("RENAME %d %s %s", dbIndex, oldKey, newKey)

	return nil
}

// Type returns the (Redis) type of the value stored at key
func (s *Store) Type(dbIndex int, key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	// verify if key exists
	if val, exists := s.data[dbIndex][key]; exists {
		switch val.Type {
		case TypeString:
			return "string"
		case TypeList:
			return "list"
		case TypeHash:
			return "hash"
		case TypeSet:
			return "set"
		case TypeZSet:
			return "zset"
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

func (s *Store) Scan(dbIndex int, cursor int, pattern string, count int) (int, []string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	allKeys := make([]string, 0, len(s.data[dbIndex]))
	for key := range s.data[dbIndex] {
		// if s.isExpired(dbIndex, key) {
		// 	continue
		// }
		value, ok := s.data[dbIndex][key]
		if ok && value.IsExpired() {
			continue
		}
		allKeys = append(allKeys, key)
	}
	if cursor < 0 || cursor >= len(allKeys) {
		return 0, []string{}, nil
	}
	if count <= 0 {
		count = 10 // default count
	}

	start := cursor
	end := cursor + count
	if end > len(allKeys) {
		end = len(allKeys)
	}
	keySlice := allKeys[start:end]
	var matchedKeys []string
	if pattern != "" && pattern != "*" {
		regexPattern := "^" + strings.ReplaceAll(strings.ReplaceAll(pattern, "?", "."), "*", ".*") + "$"
		re, err := regexp.Compile(regexPattern)
		if err != nil {
			return 0, nil, err
		}

		for _, key := range keySlice {
			if re.MatchString(key) {
				matchedKeys = append(matchedKeys, key)
			}
		}
	} else {
		matchedKeys = keySlice
	}

	var nextCursor int
	if end >= len(allKeys) {
		nextCursor = 0
	} else {
		nextCursor = end
	}

	return nextCursor, matchedKeys, nil
}
