package server

import (
	"fmt"
	"strings"

	"github.com/andrelcunha/goodiesdb/internal/protocol"
)

// Info returns server info
func (s *Server) Info() protocol.BulkString {
	s.mu.Lock()
	defer s.mu.Unlock()
	//
	var b strings.Builder
	b.WriteString("# Server\n")
	b.WriteString(fmt.Sprintf("version:%s\n", s.config.Version))
	b.WriteString(fmt.Sprintf("uptime_in_seconds:%d\n", 1000))
	b.WriteString(fmt.Sprintf("connected_clients:%d\n", 0))
	bytArr := []byte(b.String())
	fmt.Println("Sending info: ", b.String())
	return protocol.BulkString(bytArr)
}

// Ping returns pong
func (s *Server) Ping() protocol.SimpleString {
	return "PONG"
}

// Echo returns the message
func (s *Server) Echo(message string) protocol.SimpleString {
	return protocol.SimpleString(message)
}
