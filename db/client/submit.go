package client

import (
	"context"
	"fmt"
	"io"
	"strings"

	"google.golang.org/grpc"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/db/api"
)

// Submit submits the transaction to the endpoint.
func (c *Client) Submit(ctx context.Context, tx *api.Transaction) (uuid string, err error) {
	res, err := c.client.Submit(ctx, tx)
	if err != nil {
		return
	}

	uuid = res.Uuid
	return
}

func (c *Client) processGeneric2(op string) func(arg string) {
	return func(arg string) {
		arg1, arg2, err := split2args(arg)
		if err != nil {
			fmt.Println(op, "function expects two arguments: (key, data)")
			return
		}
		tx := &api.Transaction{
			Operations: []*db.Operation{{
				Key:  arg1,
				Op:   db.Operation_Op(db.Operation_Op_value[op]),
				Data: []byte(arg2),
			}},
			Policy: c.policy,
		}

		ctx, done := c.ctx()
		defer done()

		uuid, err := c.Submit(ctx, tx)
		if err != nil {
			fmt.Println("Error:", grpc.ErrorDesc(err))
			return
		}

		fmt.Println("Transaction:", uuid)
	}
}

func split2args(arg string) (arg1, arg2 string, err error) {
	args := strings.SplitN(arg, " ", 2)
	if len(args) < 2 {
		return "", "", io.ErrUnexpectedEOF
	}

	return args[0], args[1], nil
}
