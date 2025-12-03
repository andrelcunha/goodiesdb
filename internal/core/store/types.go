package store

import (
	"fmt"
	"time"

	"github.com/andrelcunha/goodiesdb/internal/protocol"
)

type ValueType byte

const (
	TypeString ValueType = iota
	TypeList
	TypeHash
	TypeSet
	TypeZSet
	TypeNull
)

type Value struct {
	Type      ValueType
	Data      interface{}
	ExpiresAt *time.Time
}

var ErrWrongType = fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
var ErrNotInteger = fmt.Errorf("ERR value is not an integer or out of range")

func NewStringValue(val string) *Value {
	return &Value{
		Type: TypeString,
		Data: val,
	}
}

func NewListValue(val []any) *Value {
	return &Value{
		Type: TypeList,
		Data: val,
	}
}

func NewHashValue(val map[string]any) *Value {
	return &Value{
		Type: TypeHash,
		Data: val,
	}
}

func NewSetValue(val map[string]struct{}) *Value {
	return &Value{
		Type: TypeSet,
		Data: val,
	}
}

func NewZSetValue(val map[string]float64) *Value {
	return &Value{
		Type: TypeZSet,
		Data: val,
	}
}

func (v *Value) AsString() (string, error) {
	if v.Type != TypeString {
		return "", ErrWrongType
	}
	str, ok := v.Data.(string)
	if !ok {
		return "", ErrWrongType
	}
	return str, nil
}

func (v *Value) AsList() ([]any, error) {
	if v.Type != TypeList {
		return nil, ErrWrongType
	}
	list, ok := v.Data.([]any)
	if !ok {
		return nil, ErrWrongType
	}
	return list, nil
}

func (v *Value) AsHash() (map[string]any, error) {
	if v.Type != TypeHash {
		return nil, ErrWrongType
	}
	hash, ok := v.Data.(map[string]any)
	if !ok {
		return nil, ErrWrongType
	}
	return hash, nil
}

func (v *Value) AsSet() (map[string]struct{}, error) {
	if v.Type != TypeSet {
		return nil, ErrWrongType
	}
	set, ok := v.Data.(map[string]struct{})
	if !ok {
		return nil, ErrWrongType
	}
	return set, nil
}

func (v *Value) AsZSet() (map[string]float64, error) {
	if v.Type != TypeZSet {
		return nil, ErrWrongType
	}
	zset, ok := v.Data.(map[string]float64)
	if !ok {
		return nil, ErrWrongType
	}
	return zset, nil
}

func (v *Value) IsExpired() bool {
	if v.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*v.ExpiresAt)
}

func (v *Value) SetExpiration(ttl time.Duration) {
	expiry := time.Now().Add(ttl)
	v.ExpiresAt = &expiry
}

func (v *Value) ToRESP() (protocol.RESPValue, error) {
	switch v.Type {
	case TypeString:
		str, _ := v.AsString()
		return protocol.BulkString([]byte(str)), nil
	case TypeList:
		list, _ := v.AsList()
		arr := make(protocol.Array, len(list))
		for i, item := range list {
			arr[i] = protocol.BulkString([]byte(fmt.Sprintf("%v", item)))
		}
		return arr, nil
	default:
		return protocol.Null{}, nil
	}
}
