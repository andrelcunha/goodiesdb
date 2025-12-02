package resp2

import (
	"bufio"
	"fmt"

	"github.com/andrelcunha/goodiesdb/internal/protocol"
)

// Implement the protocol.Protocol interface for RESP2 here

type RESP2Protocol struct{}

func (r2 *RESP2Protocol) Parse(reader *bufio.Reader) (protocol.RESPValue, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	switch prefix {
	case '+': // Simple String
		return r2.parseSimpleString(reader)
	case '-': // Error String
		return r2.parseErrorString(reader)
	case ':': // Integer
		return r2.parseInteger(reader)
	case '$': // Bulk String
		return r2.parseBulkString(reader)
	case '*': // Array
		return r2.parseArray(reader)
	default:
		return nil, fmt.Errorf("unknown RESP2 prefix: %c", prefix)
	}
}

func (r2 *RESP2Protocol) Encode(writer *bufio.Writer, value protocol.RESPValue) error {
	switch value := value.(type) {
	case protocol.SimpleString:
		return r2.encodeSimpleString(writer, value)
	case protocol.ErrorString:
		return r2.encodeErrorString(writer, value)
	case protocol.Integer:
		return r2.encodeInteger(writer, value)
	case protocol.BulkString:
		return r2.encodeBulkString(value, writer)
	case protocol.Array:
		return r2.encodeArray(value, writer)
	}
	return fmt.Errorf("encoding for type %T not implemented", value)
}

func (r2 *RESP2Protocol) Version() string {
	return "RESP2"
}
