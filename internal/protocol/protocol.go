package protocol

import "bufio"

type Protocol interface {
	Parse(reader *bufio.Reader) (RESPValue, error)
	Encode(writer *bufio.Writer, value RESPValue) error
	EncodeNil() RESPValue
	Version() string
}
