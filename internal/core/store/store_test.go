package store

import (
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/ottermq/goodiesdb/internal/utils/slice"
)

func TestStore(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)

	s := NewStore(aofChan)
	s.Set(0, "Key1", "Value1")
	value, ok := s.Get(0, "Key1")
	if !ok {
		t.Fatalf("Failed to get key")
	}
	valStr := value.Data.(string)
	if valStr != "Value1" {
		t.Fatalf("Expected Value1, got %s", valStr)
	}

	s.Del(0, "Key1")
	_, ok = s.Get(0, "Key1")
	if ok {
		t.Fatalf("Expected key1 to be deleted")
	}
}

func TestExists(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)

	s := NewStore(aofChan)
	s.Set(0, "Key1", "Value1")
	if s.Exists(0, "Key1") == 0 {
		t.Fatalf("Expected Key1 to exist")
	}
	if s.Exists(0, "Key2") > 0 {
		t.Fatalf("Expected Key2 to not exist")
	}
}

func TestSetNX(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)

	s := NewStore(aofChan)
	if s.SetNX(0, "Key1", "Value1") == 0 {
		t.Fatalf("Expected SETNX to succeed for Key1")
	}
	if s.SetNX(0, "Key1", "Value2") > 0 {
		t.Fatalf("Expected SETNX to fail for Key1")
	}
	value, ok := s.Get(0, "Key1")
	valStr := value.Data.(string)
	if !ok || valStr != "Value1" {
		t.Fatalf("Expected Value1 for Key1, got %s", valStr)
	}
}

func TestExpire(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)

	s := NewStore(aofChan)
	s.Set(0, "Key1", "Value1")
	if !s.Expire(0, "Key1", 1*time.Second) {
		t.Fatalf("Expected Expire to succeed for Key1")
	}

	time.Sleep(2 * time.Second)
	if s.Exists(0, "Key1") > 0 {
		t.Fatalf("Expected Key1 to be expired")
	}
}

func TestIncr(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)
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
	aofChan := make(chan AOFCommand, 100)
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
	aofChan := make(chan AOFCommand, 100)
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
	aofChan := make(chan AOFCommand, 100)
	s := NewStore(aofChan)

	//test if the response is correct
	listLen, _ := s.LPush(0, "list", "value1", "value2")
	if listLen != 2 {
		t.Fatalf("Expected response to be 2, got %d", listLen)
	}

	//test if the list length is correct
	s.LPush(0, "list", "value3")
	if s.GetListLength(0, "list") != 3 {
		t.Fatalf("Expected list length to be 3, got %d", s.GetListLength(0, "list"))
	}

	//test if the list contents are correct
	list := s.GetList(0, "list")
	expected := []string{"value3", "value2", "value1"}
	listStr := make([]string, len(list))
	for i, v := range list {
		listStr[i] = v.(string)
	}
	if !slice.Equal(listStr, expected) {
		t.Fatalf("Expected list to be [value3 value2 value1], got %v", listStr)
	}
}

// test RPush
func TestRPush(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)
	s := NewStore(aofChan)

	//test if the response is correct
	listLen, _ := s.RPush(0, "list", "value1", "value2")
	if listLen != 2 {
		t.Fatalf("Expected response to be 2, got %d", listLen)
	}

	//test if the list length is correct
	s.RPush(0, "list", "value3")
	if s.GetListLength(0, "list") != 3 {
		t.Fatalf("Expected list length to be 3, got %d", s.GetListLength(0, "list"))
	}

	//test if the list contents are correct
	list := s.GetList(0, "list")
	if list[0] != "value1" || list[1] != "value2" || list[2] != "value3" {
		t.Fatalf("Expected list to be [value3 value1 value2], got %v", list)
	}
}

func TestHashOperations(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)
	s := NewStore(aofChan)

	added, err := s.HSet(0, "profile", map[string]any{
		"user_id":  "1",
		"username": "andre",
	})
	if err != nil {
		t.Fatalf("HSet failed: %v", err)
	}
	if added != 2 {
		t.Fatalf("expected HSet to add 2 fields, got %d", added)
	}

	added, err = s.HSet(0, "profile", map[string]any{
		"username": "andre-updated",
		"email":    "andre@example.com",
	})
	if err != nil {
		t.Fatalf("second HSet failed: %v", err)
	}
	if added != 1 {
		t.Fatalf("expected second HSet to add 1 field, got %d", added)
	}

	value, ok, err := s.HGet(0, "profile", "username")
	if err != nil {
		t.Fatalf("HGet failed: %v", err)
	}
	if !ok || value != "andre-updated" {
		t.Fatalf("expected updated username, got ok=%v value=%q", ok, value)
	}

	values, err := s.HMGet(0, "profile", "user_id", "missing", "email")
	if err != nil {
		t.Fatalf("HMGet failed: %v", err)
	}
	if len(values) != 3 || values[0] != "1" || values[1] != nil || values[2] != "andre@example.com" {
		t.Fatalf("unexpected HMGet result: %v", values)
	}

	keys, err := s.HKeys(0, "profile")
	if err != nil {
		t.Fatalf("HKeys failed: %v", err)
	}
	sort.Strings(keys)
	expectedKeys := []string{"email", "user_id", "username"}
	if !slice.Equal(keys, expectedKeys) {
		t.Fatalf("expected HKeys %v, got %v", expectedKeys, keys)
	}

	deleted, err := s.HDel(0, "profile", "email", "missing")
	if err != nil {
		t.Fatalf("HDel failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected HDel to delete 1 field, got %d", deleted)
	}

	length, err := s.HLen(0, "profile")
	if err != nil {
		t.Fatalf("HLen failed: %v", err)
	}
	if length != 2 {
		t.Fatalf("expected HLen 2, got %d", length)
	}
}

func TestHashOperationsWrongType(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)
	s := NewStore(aofChan)

	if _, err := s.Set(0, "plain:string", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if _, err := s.HSet(0, "plain:string", map[string]any{"field": "value"}); !errors.Is(err, ErrWrongType) {
		t.Fatalf("expected ErrWrongType from HSet, got %v", err)
	}

	if _, _, err := s.HGet(0, "plain:string", "field"); !errors.Is(err, ErrWrongType) {
		t.Fatalf("expected ErrWrongType from HGet, got %v", err)
	}
}

// test LPop
func TestLPop(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)
	s := NewStore(aofChan)

	//test if LPop returns nil when key does not exist
	// t.Log("test if LPOP returns nil when key does not exist")
	value, _ := s.LPop(0, "list", nil)
	if value != nil {
		t.Fatalf("Expected got nil, got %s", value)
	}

	s.LPush(0, "list", "value1", "value2", "value3")

	//test if LPOP returns error when called with count argument smaller than 0
	count := -1
	_, err := s.LPop(0, "list", &count)
	if err == nil {
		t.Fatalf("Expected error when calling LPOP with count smaller than 0")
	}

	//test if LPOP returns empty list when called with count = 0
	count = 0
	value, err = s.LPop(0, "list", &count)
	if (err != nil) || len(value.([]any)) != 0 {
		t.Fatalf("Expected [] (empty list), got %s, value length is %d ", value, len(value.([]any)))
	}

	// test if LPOP returns the first element as string when called with count = nil
	value, err = s.LPop(0, "list", nil)
	if (err != nil) || value.(string) != "value3" {
		t.Fatalf("Expected value1, got %s", value)
	}

	//test if LPop returns the list when called with count argument greater than list length
	count = 3
	value, err = s.LPop(0, "list", &count)
	expected := []string{"value2", "value1"}
	listStr := make([]string, len(value.([]any)))
	for i, v := range value.([]any) {
		listStr[i] = v.(string)
	}
	if (err != nil) || !slice.Equal(listStr, expected) {
		t.Fatalf("Expected [value2 value1], got %v", value)
	}

	//test if LPOP returns nil when the list is empty
	// t.Log("test if LPOP returns nil when the list is empty")
	value, _ = s.LPop(0, "list", &count)
	if value != nil {
		t.Fatalf("Expected nil, got %v", value)
	}
}

// test RPop
func TestRPop(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)
	s := NewStore(aofChan)

	//test if RPop returns nil when key does not exist
	value, _ := s.RPop(0, "list", nil)
	if value != nil {
		t.Fatalf("Expected got nil, got %s", value)
	}

	s.LPush(0, "list", "value1", "value2", "value3")

	//test if RPop returns error when called with count argument smaller than 0
	count := -1
	_, err := s.RPop(0, "list", &count)
	if err == nil {
		t.Fatalf("Expected error when calling RPop with count smaller than 0")
	}

	//test if RPop returns empty list when called with count = 0
	count = 0
	list, err := s.RPop(0, "list", &count)
	listStr := make([]string, len(list.([]any)))
	for i, v := range list.([]any) {
		listStr[i] = v.(string)
	}
	if (err != nil) || len(listStr) != 0 {
		t.Fatalf("Expected [] (empty list), got %s, value length is %d ", list, len(listStr))
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
	list, err = s.RPop(0, "list", &count)
	listStr = make([]string, len(list.([]any)))
	for i, v := range list.([]any) {
		listStr[i] = v.(string)
	}

	expected := []string{"value3", "value2"}
	if (err != nil) || !slice.Equal(listStr, expected) {
		t.Fatalf("Expected [value3 value2], got %v", listStr)
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
	aofChan := make(chan AOFCommand, 100)
	s := NewStore(aofChan)

	s.LPush(0, "list", "value1", "value2", "value3", "value4")

	// Test full range
	t.Log("test if LRange returns the full range")
	list, err := s.LRange(0, "list", 0, -1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	expected := []string{"value4", "value3", "value2", "value1"}
	listStr := make([]string, len(list))
	for i, v := range list {
		listStr[i] = v.(string)
	}
	if !slice.Equal(listStr, expected) {
		t.Fatalf("Expected %v, got %v", expected, listStr)
	}

	// Test partial range
	t.Log("test if LRange returns the partial range")
	list, err = s.LRange(0, "list", 1, 2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	expected = []string{"value3", "value2"}
	listStr = make([]string, len(list))
	for i, v := range list {
		listStr[i] = v.(string)
	}
	if !slice.Equal(listStr, expected) {
		t.Fatalf("Expected %v, got %v", expected, listStr)
	}
}

// Test Rename
func TestRename(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)
	s := NewStore(aofChan)

	// test if Rename returns nil when key does not exist
	s.Rename(0, "key1", "key2")
	value, ok := s.Get(0, "key2")
	if ok {
		t.Fatalf("Expected ok equal false, got %v", ok)
	}
	if value != nil {
		t.Fatalf("Expected nil, got %v", value)
	}

	// test if Rename does not rename when key exists
	t.Log("test if Rename does not rename when key exists")
	s.Set(0, "key1", "value1")
	s.Rename(0, "key1", "key2")
	value, ok = s.Get(0, "key2")
	if !ok {
		t.Fatalf("Expected ok equal true, got %v", ok)
	}
	valStr := value.Data.(string)
	if valStr != "value1" {
		t.Fatalf("Expected value1, got %s", valStr)
	}
	s.Del(0, "key1")
	s.Del(0, "key2")

	// test if Rename does not rename when key exists and new key already exists
	t.Log("test if Rename does not rename when key exists and new key already exists")
	s.Set(0, "key1", "value1")
	s.Set(0, "key2", "value2")
	s.Rename(0, "key1", "key2")
	value, ok = s.Get(0, "key2")
	if !ok {
		t.Fatalf("Expected ok equal true, got %v", ok)
	}
	valStr = value.Data.(string)
	if valStr != "value1" {
		t.Fatalf("Expected value1, got %s", valStr)
	}
}

// Test Type
func TestType(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)
	s := NewStore(aofChan)
	dbIndex := 0

	// Arrange
	s.Set(dbIndex, "myString", "value1")
	myList := []string{"one", "two", "three"}
	mySlice := make([]any, len(myList))
	for i, v := range myList {
		mySlice[i] = v
	}
	s.RPush(dbIndex, "myList", mySlice...)

	// int
	s.SetRawValue(dbIndex, "myInt", 123)

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

	// test if an integer is string
	itype := s.Type(dbIndex, "myInt")
	if itype != "string" {
		t.Logf("expected 'string', got '%s'", itype)
		t.Fail()
	}
}

// Test Keys
func TestKeys(t *testing.T) {
	aofChan := make(chan AOFCommand, 100)
	s := NewStore(aofChan)
	indexDb := 0

	s.Set(indexDb, "key1", "value1")
	s.Set(indexDb, "key2", "value2")
	list1 := []string{"one", "two", "tree"}
	mySlice := make([]any, len(list1))
	for i, v := range list1 {
		mySlice[i] = v
	}
	s.RPush(indexDb, "list1", mySlice...)

	keys, err := s.Keys(indexDb, "*")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	expeted := []string{"key1", "key2", "list1"}
	if !slice.Equal(keys, expeted) {
		t.Logf("expected %v, got %v", expeted, keys)
	}
}
