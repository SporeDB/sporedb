package client

import (
	"context"
	"fmt"
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
		args := strings.SplitN(arg, " ", 2)
		if len(args) < 2 {
			fmt.Println(op + " function expects two arguments: (key, data)")
			return
		}

		tx := &api.Transaction{
			Operations: []*db.Operation{{
				Key:  args[0],
				Op:   db.Operation_Op(db.Operation_Op_value[op]),
				Data: []byte(args[1]),
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
