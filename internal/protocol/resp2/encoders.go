package resp2

import (
	"bufio"
	"fmt"

	"github.com/andrelcunha/goodiesdb/internal/protocol"
)

func (*RESP2Protocol) encodeSimpleString(writer *bufio.Writer, value protocol.SimpleString) error {
	_, err := writer.WriteString("+" + string(value) + "\r\n")
	return err
}

func (*RESP2Protocol) encodeErrorString(writer *bufio.Writer, value protocol.ErrorString) error {
	_, err := writer.WriteString("-" + string(value) + "\r\n")
	return err
}

func (*RESP2Protocol) encodeInteger(writer *bufio.Writer, value protocol.Integer) error {
	_, err := writer.WriteString(":" + fmt.Sprintf("%d", value) + "\r\n")
	return err
}

func (*RESP2Protocol) encodeBulkString(value protocol.BulkString, writer *bufio.Writer) error {
	bs := value
	if bs == nil { // Null Bulk String -- RESP2 representation
		_, err := writer.WriteString("$-1\r\n")
		return err
	}
	_, err := writer.WriteString("$" + fmt.Sprintf("%d", len(bs)) + "\r\n")
	if err != nil {
		return err
	}
	_, err = writer.Write(bs)
	if err != nil {
		return err
	}
	_, err = writer.WriteString("\r\n")
	return err
}

func (r2 *RESP2Protocol) encodeArray(value protocol.Array, writer *bufio.Writer) error {
	_, err := writer.WriteString("*" + fmt.Sprintf("%d", len(value)) + "\r\n")
	if err != nil {
		return err
	}
	for _, item := range value {
		err := r2.Encode(writer, item)
		if err != nil {
			return err
		}
	}
	return nil
}
