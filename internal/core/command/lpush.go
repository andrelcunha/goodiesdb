package command

import "github.com/ottermq/goodiesdb/internal/protocol"

type LPushCommand struct{}

func NewLPushCommand() *LPushCommand {
	return &LPushCommand{}
}

func (c *LPushCommand) Name() string {
	return "LPUSH"
}

func (c *LPushCommand) RequiresAuth() bool {
	return false // TODO: refactor authentication
}

func (c *LPushCommand) Validate(args []string) error {
	if len(args) < 2 {
		return ErrWrongNumberOfArguments
	}
	return nil
}

func (c *LPushCommand) Execute(ctx *Context, args []string) (protocol.RESPValue, error) {
	values := make([]any, len(args)-1)
	for i := 1; i < len(args); i++ {
		values[i-1] = args[i]
	}
	length, err := ctx.Store.LPush(ctx.DBIndex, args[0], values...)
	if err != nil {
		return protocol.ErrorString(err.Error()), nil
	}
	return protocol.Integer(int64(length)), nil
}
