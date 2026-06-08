package command

import "github.com/ottermq/goodiesdb/internal/protocol"

type RPushCommand struct{}

func NewRPushCommand() *RPushCommand {
	return &RPushCommand{}
}

func (c *RPushCommand) Name() string {
	return "RPUSH"
}

func (c *RPushCommand) RequiresAuth() bool {
	return false // TODO: refactor authentication
}

func (c *RPushCommand) Validate(args []string) error {
	if len(args) < 2 {
		return ErrWrongNumberOfArguments
	}
	return nil
}

func (c *RPushCommand) Execute(ctx *Context, args []string) (protocol.RESPValue, error) {
	values := make([]any, len(args)-1)
	for i := 1; i < len(args); i++ {
		values[i-1] = args[i]
	}
	length, err := ctx.Store.RPush(ctx.DBIndex, args[0], values...)
	if err != nil {
		return protocol.ErrorString(err.Error()), nil
	}
	return protocol.Integer(int64(length)), nil
}
