package aof

import (
	"strconv"
	"time"

	"github.com/andrelcunha/goodiesdb/internal/core/store"
)

func aofRename(parts []string, s *store.Store, dbIndex int) {
	if len(parts) == 4 {
		s.Rename(dbIndex, parts[2], parts[3])
	}
}

func aofLTrim(parts []string, s *store.Store, dbIndex int) {
	if len(parts) == 5 {
		start, _ := strconv.Atoi(parts[3])
		stop, _ := strconv.Atoi(parts[4])

		s.LTrim(dbIndex, parts[2], start, stop)
	}
}

func aofRpop(parts []string, s *store.Store, dbIndex int) {
	if len(parts) == 4 {
		count, err := strconv.Atoi(parts[3])
		if err == nil {
			s.RPop(dbIndex, parts[2], &count)
		}
	}
}

func aofLPop(parts []string, s *store.Store, dbIndex int) {
	if len(parts) == 4 {
		count, err := strconv.Atoi(parts[3])
		if err == nil {
			s.LPop(dbIndex, parts[2], &count)
		}
	}
}

func aofRPush(parts []string, s *store.Store, dbIndex int) {
	if len(parts) >= 4 {
		s.RPush(dbIndex, parts[2], parts[3:]...)
	}
}

func aofLPush(parts []string, s *store.Store, dbIndex int) {
	if len(parts) >= 4 {
		s.LPush(dbIndex, parts[2], parts[3:]...)
	}
}

func aofExpire(parts []string, s *store.Store, dbIndex int) {
	if len(parts) == 4 {
		key := parts[2]
		ttl, err := strconv.Atoi(parts[3])
		if err == nil {
			duration := time.Duration(ttl) * time.Second
			s.Expire(dbIndex, key, duration)
		}
	}
}

func aofSetNX(parts []string, s *store.Store, dbIndex int) {
	if len(parts) == 4 {
		s.SetNX(dbIndex, parts[2], parts[3])
	}
}

func aofDel(parts []string, s *store.Store, dbIndex int) {
	if len(parts) == 3 {
		s.Del(dbIndex, parts[2])
	}
}

func aofSet(parts []string, s *store.Store, dbIndex int) {
	if len(parts) == 4 {
		s.Set(dbIndex, parts[2], parts[3])
	}
}
