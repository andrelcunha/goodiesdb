package store

import (
	"testing"
	"time"

	"com.github.andrelcunha.goodiesdb/internal/utils/slice"
)

func TestStore(t *testing.T) {
	aofChan := make(chan string, 100)

	s := NewStore(aofChan)
	s.Set(0, "Key1", "Value1")
	value, ok := s.Get(0, "Key1")
	if !ok {
		t.Fatalf("Failed to get key")
	}
	if value != "Value1" {
		t.Fatalf("Expected Value1, got %s", value)
	}

	s.Del(0, "Key1")
	_, ok = s.Get(0, "Key1")
	if ok {
		t.Fatalf("Expected key1 to be deleted")
	}
}

func TestExists(t *testing.T) {
	aofChan := make(chan string, 100)

	s := NewStore(aofChan)
	s.Set(0, "Key1", "Value1")
	if !s.Exists(0, "Key1") {
		t.Fatalf("Expected Key1 to exist")
	}
	if s.Exists(0, "Key2") {
		t.Fatalf("Expected Key2 to not exist")
	}
}

func TestSetNX(t *testing.T) {
	aofChan := make(chan string, 100)

	s := NewStore(aofChan)
	if !s.SetNX(0, "Key1", "Value1") {
		t.Fatalf("Expected SETNX to succeed for Key1")
	}
	if s.SetNX(0, "Key1", "Value2") {
		t.Fatalf("Expected SETNX to fail for Key1")
	}
	value, ok := s.Get(0, "Key1")
	if !ok || value != "Value1" {
		t.Fatalf("Expected Value1, got %s", value)
	}
}

func TestExpire(t *testing.T) {
	aofChan := make(chan string, 100)

	s := NewStore(aofChan)
	s.Set(0, "Key1", "Value1")
	if !s.Expire(0, "Key1", 1*time.Second) {
		t.Fatalf("Expected Expire to succeed for Key1")
	}

	time.Sleep(2 * time.Second)
	if s.Exists(0, "Key1") {
		t.Fatalf("Expected Key1 to be expired")
	}
}

func TestIncr(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)

	newValue, err := s.Incr(0, "counter")
	if err != nil {
		t.Fatalf("INCR failed: %v", err)
	}
	// test if value is created and set as '0'++
	if newValue != 1 {
		t.Fatalf("expected 1, got %d", newValue)
	}

	// test if value is incremented
	newValue, err = s.Incr(0, "counter")
	if err != nil {
		t.Fatalf("INCR failed: %v", err)
	}
	if newValue != 2 {
		t.Fatalf("expected 2, got %d", newValue)
	}
}

func TesDecr(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)

	newValue, err := s.Decr(0, "counter")
	if err != nil {
		t.Fatalf("DECR failed: %v", err)
	}
	// test if value is created and set as '0'--
	if newValue != -1 {
		t.Fatalf("expected -1, got %d", newValue)
	}

	// test if value is incremented
	newValue, err = s.Incr(0, "counter")
	if err != nil {
		t.Fatalf("DECR failed: %v", err)
	}
	if newValue != -2 {
		t.Fatalf("expected -2, got %d", newValue)
	}
}

// test Ttl
func TestTtl(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)

	s.Set(0, "Key1", "Value1")
	if !s.Expire(0, "Key1", 4*time.Second) {
		t.Fatalf("Expected Expire to succeed for Key1")
	}
	time.Sleep(1 * time.Second)

	// Test that TTL returns the correct remaining time
	ttl, err := s.TTL(0, "Key1")
	if err != nil {
		t.Fatalf("Expected TTL to succeed for Key1")
	}
	if ttl != 2 {
		t.Fatalf("Expected TTL to be 2 seconds, got %v", ttl)
	}

	time.Sleep(3 * time.Second)

	// Test that TTL returns -2 for expired key
	ttl, err = s.TTL(0, "Key1")
	if err != nil {
		t.Fatalf("Expected TTL to succeed for Key1")
	}
	if ttl != 0 {
		t.Fatalf("Expected TTL to be -2, got %v", ttl)
	}

	s.Set(0, "Key2", "Value2")
	ttl, err = s.TTL(0, "Key2")
	if err != nil {
		t.Fatalf("Expected TTL to succeed for Key2")
	}
	if ttl != -1 {
		t.Fatalf("Expected TTL to be -1, got %v", ttl)
	}

	s.Del(0, "Key2")
	ttl, err = s.TTL(0, "Key2")
	if err != nil {
		t.Fatalf("Expected TTL to succeed for Key2")
	}
	if ttl != -2 {
		t.Fatalf("Expected TTL to be -2, got %v", ttl)
	}
}

// test LPush
func TestLPush(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)

	//test if the response is correct
	listLen := s.LPush(0, "list", "value1", "value2")
	if listLen != 2 {
		t.Fatalf("Expected response to be 2, got %d", listLen)
	}

	//test if the list length is correct
	s.LPush(0, "list", "value3")
	list, _ := s.data[0]["list"].([]string)
	if len(list) != 3 {
		t.Fatalf("Expected list length to be 3, got %d", len(list))
	}

	//test if the list contents are correct
	expected := []string{"value3", "value2", "value1"}
	if !slice.Equal(list, expected) {
		t.Fatalf("Expected list to be [value3 value2 value1], got %v", list)
	}
}

// test RPush
func TestRPush(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)

	//test if the response is correct
	listLen := s.RPush(0, "list", "value1", "value2")
	if listLen != 2 {
		t.Fatalf("Expected response to be 2, got %d", listLen)
	}

	//test if the list length is correct
	s.RPush(0, "list", "value3")
	list, _ := s.data[0]["list"].([]string)
	if len(list) != 3 {
		t.Fatalf("Expected list length to be 3, got %d", len(list))
	}

	//test if the list contents are correct
	if list[0] != "value1" || list[1] != "value2" || list[2] != "value3" {
		t.Fatalf("Expected list to be [value3 value1 value2], got %v", list)
	}
}

// test LPop
func TestLPop(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)

	//test if LPop returns nil when key does not exist
	t.Log("test if LPOP returns nil when key does not exist")
	value, _ := s.LPop(0, "list", nil)
	if value != nil {
		t.Fatalf("Expected got nil, got %s", value)
	}

	s.LPush(0, "list", "value1", "value2", "value3")

	//test if LPOP returns error when called with count argument smaller than 0
	t.Log("test if LPOP returns error when called with count argument smaller than 0")
	count := -1
	_, err := s.LPop(0, "list", &count)
	if err == nil {
		t.Fatalf("Expected error when calling LPOP with count smaller than 0")
	}

	//test if LPOP returns empty list when called with count = 0
	t.Log("test if LPOP returns empty list when called with count = 0")
	count = 0
	value, err = s.LPop(0, "list", &count)
	if (err != nil) || len(value.([]string)) != 0 {
		t.Fatalf("Expected [] (empty list), got %s, value length is %d ", value, len(value.([]string)))
	}

	// test if LPOP returns the first element as string when called with count = nil
	t.Log("test if LPOP returns the first element as string when called with count = nil")
	value, err = s.LPop(0, "list", nil)
	if (err != nil) || value.(string) != "value3" {
		t.Fatalf("Expected value1, got %s", value)
	}

	//test if LPop returns the list when called with count argument greater than list length
	t.Log("test if LPOP returns the list when called with count argument greater than list length")
	count = 3
	value, err = s.LPop(0, "list", &count)
	expected := []string{"value2", "value1"}
	if (err != nil) || !slice.Equal(value.([]string), expected) {
		t.Fatalf("Expected [value2 value1], got %v", value)
	}

	//test if LPOP returns nil when the list is empty
	t.Log("test if LPOP returns nil when the list is empty")
	value, _ = s.LPop(0, "list", &count)
	if value != nil {
		t.Fatalf("Expected nil, got %v", value)
	}
}

// test RPop
func TestRPop(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)

	//test if RPop returns nil when key does not exist
	t.Log("test if RPop returns nil when key does not exist")
	value, _ := s.RPop(0, "list", nil)
	if value != nil {
		t.Fatalf("Expected got nil, got %s", value)
	}

	s.LPush(0, "list", "value1", "value2", "value3")

	//test if RPop returns error when called with count argument smaller than 0
	t.Log("test if RPop returns error when called with count argument smaller than 0")
	count := -1
	_, err := s.RPop(0, "list", &count)
	if err == nil {
		t.Fatalf("Expected error when calling RPop with count smaller than 0")
	}

	//test if RPop returns empty list when called with count = 0
	t.Log("test if RPop returns empty list when called with count = 0")
	count = 0
	value, err = s.RPop(0, "list", &count)
	if (err != nil) || len(value.([]string)) != 0 {
		t.Fatalf("Expected [] (empty list), got %s, value length is %d ", value, len(value.([]string)))
	}
	s.Del(0, "list")

	// test if RPop returns the first element as string when called with count = nil
	s.LPush(0, "list", "value1", "value2", "value3")
	t.Log("test if RPop returns the last element as string when called with count = nil")
	value, err = s.RPop(0, "list", nil)
	if (err != nil) || value == nil || value.(string) != "value1" {
		t.Fatalf("Expected value1, got %s", value)
	}

	//test if RPop returns the list when called with count argument greater than list length
	t.Log("test if RPop returns the list when called with count argument greater than list length")
	count = 3
	value, err = s.RPop(0, "list", &count)
	expected := []string{"value3", "value2"}
	if (err != nil) || !slice.Equal(value.([]string), expected) {
		t.Fatalf("Expected [value3 value2], got %v", value)
	}

	//test if RPop returns nil when the list is empty
	t.Log("test if RPop returns nil when the list is empty")
	value, _ = s.RPop(0, "list", &count)
	if value != nil {
		t.Fatalf("Expected nil, got %v", value)
	}
}

// test LRange
func TestLRange(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)

	s.LPush(0, "list", "value1", "value2", "value3", "value4")

	// Test full range
	t.Log("test if LRange returns the full range")
	list, err := s.LRange(0, "list", 0, -1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	expected := []string{"value4", "value3", "value2", "value1"}
	if !slice.Equal(list, expected) {
		t.Fatalf("Expected %v, got %v", expected, list)
	}

	// Test partial range
	t.Log("test if LRange returns the partial range")
	list, err = s.LRange(0, "list", 1, 2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	expected = []string{"value3", "value2"}
	if !slice.Equal(list, expected) {
		t.Fatalf("Expected %v, got %v", expected, list)
	}
}

// Test Rename
func TestRename(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)

	// test if Rename returns error when key does not exist
	t.Log("test if Rename returns error when key does not exist")
	s.Rename(0, "key1", "key2")
	_, ok := s.Get(0, "key2")
	if ok {
		t.Fatalf("Expected ok equal false, got %v", ok)
	}

	// test if Rename does not rename when key exists
	t.Log("test if Rename does not rename when key exists")
	s.Set(0, "key1", "value1")
	s.Rename(0, "key1", "key2")
	value, _ := s.Get(0, "key2")
	if value != "value1" {
		t.Fatalf("Expected value1, got %s", value)
	}
	s.Del(0, "key1")
	s.Del(0, "key2")

	// test if Rename does not rename when key exists and new key already exists
	t.Log("test if Rename does not rename when key exists and new key already exists")
	s.Set(0, "key1", "value1")
	s.Set(0, "key2", "value2")
	s.Rename(0, "key1", "key2")
	value, _ = s.Get(0, "key2")
	if value != "value1" {
		t.Fatalf("Expected value1, got %s", value)
	}
}

// Test Type
func TestType(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)
	dbIndex := 0

	// Arrange
	s.Set(dbIndex, "myString", "value1")
	myList := []string{"one", "two", "three"}
	s.RPush(dbIndex, "myList", myList...)

	// int
	s.Lock()
	s.data[dbIndex]["myInt"] = 123
	s.mu.Unlock()

	// test if myString is a string
	stype := s.Type(dbIndex, "myString")
	if stype != "string" {
		t.Logf("expected 'string', got %s", stype)
		t.Fail()
	}

	// test if myList is a list
	ltype := s.Type(dbIndex, "myList")
	if ltype != "list" {
		t.Logf("expected 'list', got '%s'", ltype)
		t.Fail()
	}

	// test if a non-existing key is type 'none'
	ntype := s.Type(dbIndex, "other")
	if ntype != "none" {
		t.Logf("expected 'none', got '%s'", ntype)
		t.Fail()
	}

	// test if an integer is none
	itype := s.Type(dbIndex, "myInt")
	if itype != "none" {
		t.Logf("expected 'none', got '%s'", itype)
		t.Fail()
	}
}

// Test Keys
func TestKeys(t *testing.T) {
	aofChan := make(chan string, 100)
	s := NewStore(aofChan)
	indexDb := 0

	s.Set(indexDb, "key1", "value1")
	s.Set(indexDb, "key2", "value2")
	list1 := []string{"one", "two", "tree"}
	s.RPush(indexDb, "list1", list1...)

	keys, err := s.Keys(indexDb, "*")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	expeted := []string{"key1", "key2", "list1"}
	if !slice.Equal(keys, expeted) {
		t.Logf("expected %v, got %v", expeted, keys)
	}
}
