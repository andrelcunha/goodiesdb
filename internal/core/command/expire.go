package command

import (
	"time"

	"github.com/ottermq/goodiesdb/internal/protocol"
)

type ExpireCommand struct{}

func NewExpireCommand() *ExpireCommand {
	return &ExpireCommand{}
}

func (c *ExpireCommand) Name() string {
	return "EXPIRE"
}

func (c *ExpireCommand) RequiresAuth() bool {
	return false // TODO: refactor authentication
}

func (c *ExpireCommand) Validate(args []string) error {
	return requireExactArgs(args, 2)
}

func (c *ExpireCommand) Execute(ctx *Context, args []string) (protocol.RESPValue, error) {
	ttl, err := parseIntArg(args[1], "invalid expire time in 'expire' command")
	if err != nil {
		return nil, err
	}
	if ttl < 0 {
		return protocol.ErrorString("ERR invalid expire time in 'expire' command"), nil
	}

	if ctx.Store.Expire(ctx.DBIndex, args[0], time.Duration(ttl)*time.Second) {
		return protocol.Integer(1), nil
	}

	return protocol.Integer(0), nil
}
