package server

import (
	"fmt"
	"net"
	"path/filepath"
	"time"

	"com.github.andrelcunha.goodiesdb/internal/persistence/aof"
	"com.github.andrelcunha.goodiesdb/internal/persistence/rdb"
)

func (s *Server) isAuthenticates(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.authenticatedConnections[conn]
}

func (s *Server) getCurrentDb(conn net.Conn) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	db, ok := s.connectionDbs[conn]
	if !ok {
		db = 0
		s.connectionDbs[conn] = db
	}
	return db
}

// availableCommands returns a list of available commands
func (s *Server) availableCommands() []string {
	return []string{
		"AUTH",
		"SET",
		"GET",
		"DEL",
		"EXISTS",
		"SETNX",
		"EXPIRE",
		"INCR",
		"DECR",
		"TTL",
		"SELECT",
		"LPUSH",
		"RPUSH",
		"LPOP",
		"RPOP",
		"LRANGE",
		"LTRIM",
		"RENAME",
		"TYPE",
		"KEYS",
		"INFO",
		"PING",
		"ECHO",
		"QUIT",
		"FLUSHDB",
		"FLUSHALL",
	}
}

// Info returns server info
func (s *Server) Info() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("Server info:\n\n"+
		"Version: %s\n",
		s.config.Version,
	)
}

// Ping returns pong
func (s *Server) Ping() string {
	return "PONG"
}

// Echo returns the message
func (s *Server) Echo(message string) string {
	return message
}

// Quit closes the connection
func (s *Server) Quit(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintln(conn, "OK")
	delete(s.authenticatedConnections, conn)
	conn.Close()
}

// SelectDb selects the database
func (s *Server) SelectDb(conn net.Conn, dbIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if dbIndex < 0 || dbIndex >= s.store.Count() {
		return fmt.Errorf("invalid DB index")
	}
	s.connectionDbs[conn] = dbIndex
	return nil
}

func (s *Server) startRDB() {
	rdbFilepath := filepath.Join(s.dataDir, "dump.rdb")
	for {
		select {
		case <-time.After(1 * time.Minute):
			if err := rdb.SaveSnapshot(s.store, rdbFilepath); err != nil {
				fmt.Println("Error saving snapshot:", err)
			} else {
				fmt.Println("Snapshot saved successfully")
			}

		case <-s.shutdownChan:
			return
		}
	}
}

func (s *Server) recoverStore() {
	rdbFilepath := filepath.Join(s.dataDir, "dump.rdb")
	aofFilepath := filepath.Join(s.dataDir, "appendonly.aof")
	flagOk := false
	if s.config.UseRDB {
		if err := rdb.LoadSnapshot(s.store, rdbFilepath); err != nil {
			fmt.Println("No snapshot found.")
		} else {
			flagOk = true
		}
	}

	if s.config.UseAOF && !flagOk {
		if err := aof.RebuildStoreFromAOF(s.store, aofFilepath); err != nil {
			fmt.Println("Error loading from AOF:", err)

		} else {
			flagOk = true
		}
	}
	if !flagOk {
		fmt.Println("None of the recovery files are healthy. Starting with an empty store.")
	}
}

func (s *Server) asciiLogo() string {
	return `
  G)gggg                      d) ##                 D)dddd   B)bbbb   
 G)                           d)                    D)   dd  B)   bb  
G)  ggg   o)OOO   o)OOO   d)DDDD i) e)EEEEE  s)SSSS D)    dd B)bbbb   
G)    gg o)   OO o)   OO d)   DD i) e)EEEE  s)SSSS  D)    dd B)   bb  
 G)   gg o)   OO o)   OO d)   DD i) e)           s) D)    dd B)    bb 
  G)ggg   o)OOO   o)OOO   d)DDDD i)  e)EEEE s)SSSS  D)ddddd  B)bbbbb  

`
}
