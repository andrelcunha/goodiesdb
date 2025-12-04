package resp2

import (
	"bufio"
	"fmt"

	"github.com/andrelcunha/goodiesdb/internal/protocol"
)

func (*RESP2Protocol) parseSimpleString(reader *bufio.Reader) (protocol.SimpleString, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return protocol.SimpleString(line[:len(line)-2]), nil
}

func (*RESP2Protocol) parseErrorString(reader *bufio.Reader) (protocol.RESPValue, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return protocol.ErrorString(line[:len(line)-2]), nil
}

func (*RESP2Protocol) parseInteger(reader *bufio.Reader) (protocol.RESPValue, error) {
	var value int64
	_, err := fmt.Fscanf(reader, "%d\r\n", &value)
	if err != nil {
		return nil, err
	}
	return protocol.Integer(value), nil
}

func (*RESP2Protocol) parseBulkString(reader *bufio.Reader) (protocol.RESPValue, error) {
	var length int
	_, err := fmt.Fscanf(reader, "%d\r\n", &length)
	if err != nil {
		return nil, err
	}
	if length == -1 {
		return protocol.BulkString(nil), nil // Null Bulk String
	}
	data := make([]byte, length+2)
	_, err = reader.Read(data)
	if err != nil {
		return nil, err
	}
	return protocol.BulkString(data[:length]), nil
}

func (r2 *RESP2Protocol) parseArray(reader *bufio.Reader) (protocol.RESPValue, error) {
	var count int
	_, err := fmt.Fscanf(reader, "%d\r\n", &count)
	if err != nil {
		return nil, err
	}
	if count == -1 {
		return protocol.Array(nil), nil // Null Array
	}
	array := make(protocol.Array, count)
	for i := 0; i < count; i++ {
		value, err := r2.Parse(reader)
		if err != nil {
			return nil, err
		}
		array[i] = value
	}
	return array, nil
}
