package protocol

type RESPValue interface{}

// RESP2 types
type SimpleString string
type ErrorString string
type Integer int64
type BulkString []byte
type Array []RESPValue

// RESP3 types
type Map map[RESPValue]RESPValue
type Set []RESPValue
type Boolean bool
type Double float64
type BigNumber string
type Null struct{}
type Push []RESPValue
