package store

import (
	"fmt"
	"strconv"
	"strings"
)

// Set sets the value for a key
// Consider ret
func (s *Store) Set(dbIndex int, key string, rawValue any, args ...string) (bool, error) {
	setOptions, err := parseSetOptions(args)
	if err != nil {
		return false, err
	}
	// Handle NX and XX options
	if setOptions.NX && s.Exists(dbIndex, key) > 0 {
		return false, nil
	}
	if setOptions.XX && s.Exists(dbIndex, key) == 0 {
		return false, nil
	}
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
	s.data[dbIndex][key] = value
	return true, nil
}

type SetOptions struct {
	NX bool // Only set if key does not exist
	XX bool // Only set if key exists
	EX int  // Expire time in seconds
	PX int  // Expire time in milliseconds
}

func parseSetOptions(args []string) (*SetOptions, error) {
	options := &SetOptions{}
	i := 0
	for i < len(args) {
		switch strings.ToUpper(args[i]) {
		case "NX":
			options.NX = true
			i++
		case "XX":
			options.XX = true
			i++
		case "EX":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for EX option")
			}
			seconds, err := strconv.Atoi(args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid value for EX option")
			}
			options.EX = seconds
			i += 2
		case "PX":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for PX option")
			}
			milliseconds, err := strconv.Atoi(args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid value for PX option")
			}
			options.PX = milliseconds
			i += 2
		default:
			return nil, fmt.Errorf("unknown option: %s", args[i])
		}
	}
	if options.NX && options.XX {
		return nil, fmt.Errorf("ERR syntax error")
	}
	return options, nil
}
