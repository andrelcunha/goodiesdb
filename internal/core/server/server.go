package server

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andrelcunha/goodiesdb/internal/core/store"
	"github.com/andrelcunha/goodiesdb/internal/persistence/aof"
	"github.com/andrelcunha/goodiesdb/internal/persistence/rdb"
	"github.com/andrelcunha/goodiesdb/internal/protocol"
	"github.com/andrelcunha/goodiesdb/internal/protocol/resp2"
)

// Server represents a TCP server
type Server struct {
	store                    *store.Store
	config                   *Config
	mu                       sync.Mutex
	authenticatedConnections map[net.Conn]bool // TODO create a connection abstraction to hold more info
	connectionDbs            map[net.Conn]int
	shutdownChan             chan struct{}
	dataDir                  string
	Protocol                 protocol.Protocol
}

// NewServer creates a new server
func NewServer(config *Config) *Server {
	// Create the data directory if it doesn't exist
	dataDir := config.DataDir
	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		os.Exit(1)
	}

	aofChan := make(chan string, 100)
	s := store.NewStore(aofChan)

	return &Server{
		store:                    s,
		config:                   config,
		authenticatedConnections: make(map[net.Conn]bool),
		connectionDbs:            make(map[net.Conn]int),
		shutdownChan:             make(chan struct{}),
		dataDir:                  config.DataDir,
		Protocol:                 &resp2.RESP2Protocol{},
	}
}

// Start starts the server
func (s *Server) Start() error {
	fmt.Println(s.asciiLogo())
	fmt.Println("Starting Redis Clone Server...")

	if s.config.UseRDB || s.config.UseAOF {
		fmt.Println("Found persistence enabled. Recovering data...")
		s.recoverStore()
	} else {
		fmt.Println("No persistence enabled. Data will not be persisted.")
	}

	if s.config.UseRDB {
		go s.startRDB()
		fmt.Println("RDB persistence enabled")
	}
	if s.config.UseAOF {
		aofFilepath := filepath.Join(s.dataDir, "appendonly.aof")
		go aof.AOFWriter(s.store.AOFChannel(), aofFilepath)
		fmt.Println("AOF persistence enabled")
	}

	// set addr string (host and port) using config
	addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	ln, err := net.Listen("tcp", addr)
	fmt.Printf("Redis Clone Server %s started on %s:%s\n", s.config.Version, s.config.Host, s.config.Port)
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		// go s.handleConnection(conn)
		go s.handleConn(conn)
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	if s.config.UseAOF {
		if s.store.AOFChannel() != nil {
			close(s.store.AOFChannel())
		}
	}

	if s.config.UseRDB {
		rdb.SaveSnapshot(s.store, "dump.rdb")
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		value, err := s.Protocol.Parse(reader)

		if err != nil {
			if err.Error() == "EOF" {
				return
			}
			reply := protocol.ErrorString(fmt.Sprintf("parse error: %v", err))
			s.Protocol.Encode(writer, reply)
			writer.Flush()
			continue
		}

		// Execute commmand
		reply, err := s.executeCommand(conn, value)
		if err != nil {
			reply := protocol.ErrorString(fmt.Sprintf("ERR %s", err.Error()))
			s.Protocol.Encode(writer, reply)
			writer.Flush()
			continue
		}

		s.Protocol.Encode(writer, reply)
		writer.Flush()
		continue
	}
}

func (s *Server) executeCommand(conn net.Conn, request protocol.RESPValue) (protocol.RESPValue, error) {
	arr, ok := request.(protocol.Array)
	if !ok {
		return protocol.ErrorString("ERR expected array"), fmt.Errorf("expected array, got %T", request)
	}

	rawParts := arr

	if len(rawParts) == 0 {
		return "", nil
	}

	parts := convertArrayToStrings(rawParts)

	fmt.Println("Executing command:", parts[0])

	// //check authentication
	// if !s.isAuthenticates(conn) && strings.ToUpper(parts[0]) != "AUTH" {
	// 	return protocol.ErrorString("NOAUTH Authentication required."), nil
	// }

	dbIndex := s.getCurrentDb(conn)

	switch strings.ToUpper(parts[0]) {

	case "AUTH":
		if len(parts) != 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'AUTH' command"), nil
		}
		if parts[1] == s.config.Password {
			s.mu.Lock()
			s.authenticatedConnections[conn] = true
			s.mu.Unlock()
			return protocol.SimpleString("OK"), nil
		} else {
			return protocol.ErrorString("ERR invalid password"), nil
		}

	case "SET":
		if len(parts) != 3 {
			return protocol.ErrorString("ERR wrong number of arguments for 'SET' command"), nil
		}
		s.store.Set(dbIndex, parts[1], parts[2])
		return protocol.SimpleString("OK"), nil

	case "GET":
		if len(parts) != 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'GET' command"), nil
		}
		value, ok := s.store.Get(dbIndex, parts[1])
		if !ok {
			return s.Protocol.EncodeNil(), nil
		}
		r, err := convertValueTypeToRESPType(value)
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		return r, nil

	case "DEL":
		if len(parts) != 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'DEL' command"), nil
		}
		s.store.Del(dbIndex, parts[1])
		return protocol.SimpleString("OK"), nil

	case "EXISTS":
		if len(parts) != 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'EXISTS' command"), nil
		}
		exists := s.store.Exists(dbIndex, parts[1])
		if exists {
			return "1", nil
		} else {
			return "0", nil
		}

	case "SETNX":
		if len(parts) != 3 {
			return protocol.ErrorString("ERR wrong number of arguments for 'SETNX' command"), nil
		}
		if s.store.SetNX(dbIndex, parts[1], parts[2]) {
			return "1", nil
		} else {
			return "0", nil
		}

	case "EXPIRE":
		if len(parts) != 3 {
			return protocol.ErrorString("ERR wrong number of arguments for 'EXPIRE' command"), nil
		}
		key := parts[1]
		ttl, err := strconv.Atoi(parts[2])
		if err != nil {
			return protocol.ErrorString("ERR invalid TTL"), nil
		}
		duration := time.Duration(ttl) * time.Second
		if s.store.Expire(dbIndex, key, duration) {
			return "1", nil
		} else {
			return "0", nil
		}

	case "INCR":
		if len(parts) != 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'INCR' command"), nil
		}
		newValue, err := s.store.Incr(dbIndex, parts[1])
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		return newValue, nil

	case "DECR":
		if len(parts) != 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'DECR' command"), nil
		}
		newValue, err := s.store.Decr(dbIndex, parts[1])
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		return newValue, nil

	case "TTL":
		if len(parts) != 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'TTL' command"), nil
		}
		ttl, err := s.store.TTL(dbIndex, parts[1])
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		return ttl, nil

	case "SELECT":
		if len(parts) != 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'SELECT' command"), nil
		}
		dbIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			return protocol.ErrorString("ERR invalid DB index"), nil
		}
		err = s.SelectDb(conn, dbIndex)
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		return "OK", nil

	case "LPUSH":
		if len(parts) < 3 {
			return protocol.ErrorString("ERR wrong number of arguments for 'LPUSH' command"), nil
		}
		slice := make([]any, len(parts)-2)
		for i := 2; i < len(parts); i++ {
			slice[i-2] = parts[i]
		}
		length := s.store.LPush(dbIndex, parts[1], slice...)
		return length, nil

	case "RPUSH":
		if len(parts) < 3 {
			return protocol.ErrorString("ERR wrong number of arguments for 'RPUSH' command"), nil
		}
		slice := make([]any, len(parts)-2)
		for i := 2; i < len(parts); i++ {
			slice[i-2] = parts[i]
		}
		length := s.store.RPush(dbIndex, parts[1], slice...)
		return length, nil

	case "LPOP":
		if len(parts) != 2 && len(parts) != 3 {
			return protocol.ErrorString("ERR wrong number of arguments for 'LPOP' command"), nil
		}
		var count *int
		if len(parts) == 3 {
			c, err := strconv.Atoi(parts[2])
			//parts[2] should be already parsed as int
			// c, ok := parts[2].(int)
			if err != nil {
				return protocol.ErrorString("ERR value is out of range, must be positive"), nil
			}
			count = &c
		}
		value, err := s.store.LPop(dbIndex, parts[1], count)
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		fmt.Fprintln(conn, value)

	case "RPOP":
		if len(parts) != 2 && len(parts) != 3 {
			return protocol.ErrorString("ERR wrong number of arguments for 'RPOP' command"), nil
		}
		var count *int
		if len(parts) == 3 {
			c, err := strconv.Atoi(parts[2])
			if err != nil {
				return protocol.ErrorString("ERR value is out of range, must be positive"), nil
			}
			count = &c
		}
		value, err := s.store.RPop(dbIndex, parts[1], count)
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		return value, nil

	case "LRANGE":
		if len(parts) != 4 {
			return protocol.ErrorString("ERR wrong number of arguments for 'LRANGE' command"), nil
		}
		start, err1 := strconv.Atoi(parts[2])
		stop, err2 := strconv.Atoi(parts[3])
		if err1 != nil || err2 != nil {
			return protocol.ErrorString("ERR value is out of range, must be positive"), nil
		}
		values, err := s.store.LRange(dbIndex, parts[1], start, stop)
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		return values, nil

	case "LTRIM":
		if len(parts) != 4 {
			return protocol.ErrorString("ERR wrong number of arguments for 'LTRIM' command"), nil
		}
		start, err1 := strconv.Atoi(parts[2])
		stop, err2 := strconv.Atoi(parts[3])
		if err1 != nil || err2 != nil {
			return protocol.ErrorString("ERR value is out of range, must be positive"), nil
		}
		err := s.store.LTrim(dbIndex, parts[1], start, stop)
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		return protocol.SimpleString("OK"), nil

	case "RENAME":
		if len(parts) != 3 {
			return protocol.ErrorString("ERR wrong number of arguments for 'RENAME' command"), nil
		}
		if err := s.store.Rename(dbIndex, parts[1], parts[2]); err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		return protocol.SimpleString("OK"), nil

	case "TYPE":
		if len(parts) != 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'TYPE' command"), nil
		}
		vtype := s.store.Type(dbIndex, parts[1])
		return protocol.SimpleString(vtype), nil

	case "KEYS":
		if len(parts) != 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'TYPE' command"), nil
		}
		pattern := parts[1]
		keys, err := s.store.Keys(dbIndex, pattern)
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}
		return keys, nil

	case "INFO":
		return s.Info(), nil

	case "PING":
		return s.Ping(), nil

	case "ECHO":
		if len(parts) < 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'ECHO' command"), nil
		}
		strs := make([]string, len(parts)-1)
		for i := 1; i < len(parts); i++ {
			strs[i-1] = fmt.Sprintf("%v", parts[i])
		}
		return s.Echo(strings.Join(strs, " ")), nil

	case "QUIT":
		s.Quit(conn)

	case "FLUSHDB":
		s.store.FlushDb(dbIndex)
		fmt.Fprintln(conn, "OK")

	case "FLUSHALL":
		s.store.FlushAll()
		fmt.Fprintln(conn, "OK")

	case "SCAN":
		if len(parts) < 2 {
			return protocol.ErrorString("ERR wrong number of arguments for 'SCAN' command"), nil
		}
		cursor, err := strconv.Atoi(parts[1])
		if err != nil {
			return protocol.ErrorString("ERR invalid cursor"), nil
		}

		pattern := "*"
		count := 10

		for i := 2; i < len(parts); i++ {
			switch strings.ToUpper(parts[i]) {
			case "MATCH":
				if i+1 >= len(parts) {
					return protocol.ErrorString("ERR syntax error"), nil
				}
				pattern = parts[i+1]
				i++
			case "COUNT":
				if i+1 >= len(parts) {
					return protocol.ErrorString("ERR syntax error"), nil
				}
				c, err := strconv.Atoi(parts[i+1])
				if err != nil || c <= 0 {
					return protocol.ErrorString("ERR invalid count"), nil
				}
				count = c
				i++
			default:
				return protocol.ErrorString("ERR syntax error"), nil
			}
		}

		newCursor, keys, err := s.store.Scan(dbIndex, cursor, pattern, count)
		if err != nil {
			return protocol.ErrorString("ERR " + err.Error()), nil
		}

		result := protocol.Array{
			protocol.BulkString(strconv.Itoa(newCursor)),
			protocol.Array(func() []protocol.RESPValue {
				arr := make([]protocol.RESPValue, len(keys))
				for i, k := range keys {
					arr[i] = protocol.BulkString(k)
				}
				return arr
			}()),
		}
		return result, nil

	default:
		return protocol.ErrorString("ERR unknown command '" + parts[0] + "'"), nil
	}
	return nil, nil
}

func convertArrayToStrings(rawParts protocol.Array) []string {
	parts := make([]string, len(rawParts))
	for i, part := range rawParts {
		switch v := part.(type) {
		case protocol.BulkString:
			parts[i] = string(v)
		case protocol.SimpleString:
			parts[i] = string(v)
		case string:
			parts[i] = v
		default:
			return nil
		}
	}
	return parts
}

func convertValueTypeToRESPType(val interface{}) (protocol.RESPValue, error) {
	value := val.(store.Value)
	switch value.Type {
	case store.TypeString:
		return protocol.SimpleString(value.Data.(string)), nil
	case store.TypeList:
		list := value.Data.([]any)
		respArray := protocol.Array{}
		for _, v := range list {
			respArray = append(respArray, protocol.SimpleString(fmt.Sprintf("%v", v)))
		}
		return respArray, nil
	default:
		return protocol.ErrorString("ERR unknown type"), nil
	}
}
