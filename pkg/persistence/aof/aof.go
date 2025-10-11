package aof

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	"com.github.andrelcunha.goodiesdb/pkg/store"
)

// AOFWriter writes commands to a file
func AOFWriter(aofChan chan string, filename string) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open AOF file: %v", err)
	}
	defer file.Close()

	for cmd := range aofChan {
		_, err := file.WriteString(cmd + "\n")
		if err != nil {
			log.Fatalf("Failed to write to AOF file: %v", err)
		}
	}
}

// RebuildStoreFromAOF rebuilds the store from the AOF file
func RebuildStoreFromAOF(s *store.Store, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create scanner to read the AOF file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cmd := scanner.Text()
		parts := strings.Split(cmd, " ")
		if len(parts) == 0 {
			continue
		}

		dbIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Printf("Invalid database index: %s", parts[1])
			continue
		}

		switch parts[0] {

		case "SET":
			aofSet(parts, s, dbIndex)

		case "DEL":
			aofDel(parts, s, dbIndex)

		case "SETNX":
			aofSetNX(parts, s, dbIndex)

		case "EXPIRE":
			aofExpire(parts, s, dbIndex)

		case "LPUSH":
			aofLPush(parts, s, dbIndex)

		case "RPUSH":
			aofRPush(parts, s, dbIndex)

		case "LPOP":
			aofLPop(parts, s, dbIndex)

		case "RPOP":
			aofRpop(parts, s, dbIndex)

		case "LTRIM":
			aofLTrim(parts, s, dbIndex)

		case "RENAME":
			aofRename(parts, s, dbIndex)

		default:
			log.Printf("Unknown command: %s", cmd)
		}
	}

	return scanner.Err()
}
