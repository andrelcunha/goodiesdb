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

	"com.github.andrelcunha.goodiesdb/pkg/persistence/aof"
	"com.github.andrelcunha.goodiesdb/pkg/persistence/rdb"
	"com.github.andrelcunha.goodiesdb/pkg/store"
)

// Server represents a TCP server
type Server struct {
	store                    *store.Store
	config                   *Config
	mu                       sync.Mutex
	authenticatedConnections map[net.Conn]bool
	connectionDbs            map[net.Conn]int
	shutdownChan             chan struct{}
	dataDir                  string
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
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			return
		}
		s.handleCommand(conn, strings.TrimSpace(line))
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

// handleCommand handles a single command
func (s *Server) handleCommand(conn net.Conn, cmd string) {
	parts := strings.Fields(cmd)

	if len(parts) == 0 {
		return
	}

	//check authentication
	if !s.isAuthenticates(conn) && parts[0] != "AUTH" {
		fmt.Fprintln(conn, "NOAUTH Authentication required.")
		return
	}

	dbIndex := s.getCurrentDb(conn)

	switch parts[0] {

	case "AUTH":
		if len(parts) != 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'AUTH' command")
			return
		}
		if parts[1] == s.config.Password {
			s.mu.Lock()
			s.authenticatedConnections[conn] = true
			s.mu.Unlock()
			fmt.Fprintln(conn, "OK")
		} else {
			fmt.Fprintln(conn, "ERR invalid password")
		}

	case "SET":
		if len(parts) != 3 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'SET' command")
			return
		}
		s.store.Set(dbIndex, parts[1], parts[2])
		fmt.Fprintln(conn, "OK")

	case "GET":
		if len(parts) != 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'GET' command")
			return
		}
		value, ok := s.store.Get(dbIndex, parts[1])
		if !ok {
			fmt.Fprintln(conn, "NULL")
		} else {
			fmt.Fprintln(conn, value)
		}

	case "DEL":
		if len(parts) != 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'DEL' command")
			return
		}
		s.store.Del(dbIndex, parts[1])
		fmt.Fprintln(conn, "OK")

	case "EXISTS":
		if len(parts) != 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'EXISTS' command")
			return
		}
		exists := s.store.Exists(dbIndex, parts[1])
		if exists {
			fmt.Fprintln(conn, 1)
		} else {
			fmt.Fprintln(conn, 0)
		}

	case "SETNX":
		if len(parts) != 3 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'SETNX' command")
			return
		}
		if s.store.SetNX(dbIndex, parts[1], parts[2]) {
			fmt.Fprintln(conn, 1)
		} else {
			fmt.Fprintln(conn, 0)
		}

	case "EXPIRE":
		if len(parts) != 3 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'EXPIRE' command")
			return
		}
		key := parts[1]
		ttl, err := strconv.Atoi(parts[2])
		if err != nil {
			fmt.Fprintln(conn, "ERR invalid TTL")
			return
		}
		duration := time.Duration(ttl) * time.Second
		if s.store.Expire(dbIndex, key, duration) {
			fmt.Fprintln(conn, 1)
		} else {
			fmt.Fprintln(conn, 0)
		}

	case "INCR":
		if len(parts) != 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'INCR' command")
			return
		}
		newValue, err := s.store.Incr(dbIndex, parts[1])
		if err != nil {
			fmt.Fprintln(conn, "ERR ", err.Error())
			return
		}
		fmt.Fprintln(conn, newValue)

	case "DECR":
		if len(parts) != 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'DECR' command")
			return
		}
		newValue, err := s.store.Decr(dbIndex, parts[1])
		if err != nil {
			fmt.Fprintln(conn, "ERR ", err.Error())
			return
		}
		fmt.Fprintln(conn, newValue)

	case "TTL":
		if len(parts) != 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'TTL' command")
			return
		}
		ttl, err := s.store.TTL(dbIndex, parts[1])
		if err != nil {
			fmt.Fprintln(conn, "ERR ", err.Error())
			return
		}
		fmt.Fprintln(conn, ttl)

	case "SELECT":
		if len(parts) != 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'SELECT' command")
			return
		}
		dbIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Fprintln(conn, "ERR invalid DB index")
			return
		}
		err = s.SelectDb(conn, dbIndex)
		if err != nil {
			fmt.Fprintln(conn, "ERR ", err.Error())
			return
		}
		fmt.Fprintln(conn, "OK")

	case "LPUSH":
		if len(parts) < 3 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'LPUSH' command")
			return
		}
		length := s.store.LPush(dbIndex, parts[1], parts[2:]...)
		fmt.Fprintln(conn, length)

	case "RPUSH":
		if len(parts) < 3 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'RPUSH' command")
			return
		}
		length := s.store.RPush(dbIndex, parts[1], parts[2:]...)
		fmt.Fprintln(conn, length)

	case "LPOP":
		if len(parts) != 2 && len(parts) != 3 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'LPOP' command")
			return
		}
		var count *int
		if len(parts) == 3 {
			c, err := strconv.Atoi(parts[2])
			if err != nil {
				fmt.Fprintln(conn, "ERR value is out of range, must be positive")
				return
			}
			count = &c
		}
		value, err := s.store.LPop(dbIndex, parts[1], count)
		if err != nil {
			fmt.Fprintln(conn, "ERR ", err.Error())
			return
		}
		fmt.Fprintln(conn, value)

	case "RPOP":
		if len(parts) != 2 && len(parts) != 3 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'RPOP' command")
			return
		}
		var count *int
		if len(parts) == 3 {
			c, err := strconv.Atoi(parts[2])
			if err != nil {
				fmt.Fprintln(conn, "ERR value is out of range, must be positive")
				return
			}
			count = &c
		}
		value, err := s.store.RPop(dbIndex, parts[1], count)
		if err != nil {
			fmt.Fprintln(conn, "ERR ", err.Error())
			return
		}
		fmt.Fprintln(conn, value)

	case "LRANGE":
		if len(parts) != 4 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'LRANGE' command")
			return
		}
		start, err1 := strconv.Atoi(parts[2])
		stop, err2 := strconv.Atoi(parts[3])
		if err1 != nil || err2 != nil {
			fmt.Fprintln(conn, "ERR value is out of range, must be positive")
			return
		}
		values, err := s.store.LRange(dbIndex, parts[1], start, stop)
		if err != nil {
			fmt.Fprintln(conn, "ERR ", err.Error())
			return
		}
		fmt.Fprintln(conn, values)

	case "LTRIM":
		if len(parts) != 4 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'LTRIM' command")
			return
		}
		start, err1 := strconv.Atoi(parts[2])
		stop, err2 := strconv.Atoi(parts[3])
		if err1 != nil || err2 != nil {
			fmt.Fprintln(conn, "ERR value is out of range, must be positive")
			return
		}
		err := s.store.LTrim(dbIndex, parts[1], start, stop)
		if err != nil {
			fmt.Fprintln(conn, "ERR ", err.Error())
			return
		}
		fmt.Fprintln(conn, "OK")

	case "RENAME":
		if len(parts) != 3 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'RENAME' command")
			return
		}
		if err := s.store.Rename(dbIndex, parts[1], parts[2]); err != nil {
			fmt.Fprintln(conn, "ERR", err.Error())
			return
		}
		fmt.Fprintln(conn, "OK")

	case "TYPE":
		if len(parts) != 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'TYPE' command")
			return
		}
		vtype := s.store.Type(dbIndex, parts[1])
		fmt.Fprintln(conn, vtype)

	case "KEYS":
		if len(parts) != 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'TYPE' command")
			return
		}
		pattern := parts[1]
		keys, err := s.store.Keys(dbIndex, pattern)
		if err != nil {
			fmt.Fprintln(conn, "ERR ", err.Error())
		}
		fmt.Fprintln(conn, keys)

	case "INFO":
		fmt.Fprintln(conn, s.Info())

	case "PING":
		fmt.Fprintln(conn, s.Ping())

	case "ECHO":
		if len(parts) < 2 {
			fmt.Fprintln(conn, "ERR wrong number of arguments for 'ECHO' command")
			return
		}
		fmt.Fprintln(conn, s.Echo(strings.Join(parts[1:], " ")))

	case "QUIT":
		s.Quit(conn)

	case "FLUSHDB":
		s.store.FlushDb(dbIndex)
		fmt.Fprintln(conn, "OK")

	case "FLUSHALL":
		s.store.FlushAll()
		fmt.Fprintln(conn, "OK")

	default:
		fmt.Fprintln(conn, "ERR unknown command '"+parts[0]+"'")
		fmt.Fprintln(conn, "Available commands: "+strings.Join(s.availableCommands(), " "))
	}
}
