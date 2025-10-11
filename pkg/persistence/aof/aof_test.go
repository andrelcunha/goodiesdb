package aof

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"com.github.andrelcunha.goodiesdb/internal/utils/slice"
	"com.github.andrelcunha.goodiesdb/pkg/store"
)

func TestRebuildStoreFromAOF(t *testing.T) {
	aofFilename := "test_appendonly.aof"
	os.Remove(aofFilename)
	aofChan := make(chan string, 100)

	// Start the AOF writer
	go AOFWriter(aofChan, aofFilename)

	// Initialize the store with AOF logging
	s := store.NewStore(aofChan)

	dbIndex := 0

	// Set and expire commands
	s.Set(dbIndex, "Key1", "Value1")
	s.Set(dbIndex, "Key2", "Value2")
	s.Expire(dbIndex, "Key1", 3*time.Second)

	// SETNX command
	s.SetNX(dbIndex, "Key3", "Value3")   // Should succeed
	s.SetNX(dbIndex, "Key3", "NewValue") // Should fail because Key3 already exists

	// List commands
	s.LPush(dbIndex, "List1", "Value1", "Value2", "Value3")
	s.RPush(dbIndex, "List1", "Value4")
	s.LPop(dbIndex, "List1", nil)
	s.RPop(dbIndex, "List1", nil)

	// List trimming commands
	s.LTrim(dbIndex, "List1", 1, 2)

	// Rename command
	s.Rename(dbIndex, "Key2", "RenamedKey")

	// Give some time for commands to be written to AOF
	time.Sleep(1 * time.Second)

	// Rebuild state from AOF
	newAofFilename := "new_test_appendonly.aof"
	os.Remove(newAofFilename)
	newAofChan := make(chan string, 100)
	go AOFWriter(newAofChan, newAofFilename)

	newStore := store.NewStore(newAofChan)

	err := RebuildStoreFromAOF(newStore, aofFilename)
	if err != nil {
		t.Fatalf("Failed to rebuild state from AOF: %v", err)
	}
	// set new aofFilename

	// Verify Key2 has been renamed to RenamedKey
	value, ok := newStore.Get(dbIndex, "RenamedKey")
	if !ok || value != "Value2" {
		t.Errorf("Expected Value2 for RenamedKey, got %s", value)
		t.Fail()
	}

	// Verify List1 contents
	list, _ := newStore.LRange(dbIndex, "List1", 0, -1)
	expectedList := []string{"Value1"}
	if !slice.Equal(list, expectedList) {
		t.Errorf("Expected %v, got %v", expectedList, list)
		t.Fail()
	}

	// Wait for the key to expire
	time.Sleep(4 * time.Second)

	// Verify Key1 exists after it expires
	if newStore.Exists(dbIndex, "Key1") {
		t.Errorf("Expected Key1 to be expired after waiting more than 3 seconds")
		t.Fail()
	}

	// Clean up the AOF file
	os.Remove(aofFilename)
	os.Remove(newAofFilename)
}

// Test aofRename
func TestAofRename(t *testing.T) {
	cmd := "RENAME 0 Key1 newName"
	parts, s, dbIndex := prepareCmdTest(cmd)

	s.Set(dbIndex, "Key1", "value1")

	aofRename(parts, s, dbIndex)
	value, ok := s.Get(dbIndex, "newName")
	if !ok || value != "value1" {
		t.Fatalf("Expeted 'value1, got %s", value)
	}
}

// Test aofLTrim
func TestAofLTrim(t *testing.T) {
	cmd := "LTRIM 0 List1 1 2"
	parts, s, dbIndex := prepareCmdTest(cmd)

	s.LPush(dbIndex, "List1", "Value1", "Value2", "Value3")
	aofLTrim(parts, s, dbIndex)
	list, _ := s.LRange(dbIndex, "List1", 0, -1)
	expectedList := []string{"Value2, Value1"}
	if slice.Equal(list, expectedList) {
		t.Logf("Expected %v, got %v", expectedList, list)
		t.Fail()
	}
}

func prepareCmdTest(cmd string) ([]string, *store.Store, int) {
	aofChan := make(chan string, 100)
	s := store.NewStore(aofChan)

	parts := strings.Split(cmd, " ")

	dbIndex, _ := strconv.Atoi(parts[1])
	return parts, s, dbIndex
}
