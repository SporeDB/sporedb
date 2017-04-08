package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"gitlab.com/SporeDB/sporedb/db/api"
	"gitlab.com/SporeDB/sporedb/db/version"
)

// Get gets the key from the endpoint.
func (c *Client) Get(ctx context.Context, key string) (value []byte, v *version.V, err error) {
	res, err := c.client.Get(ctx, &api.Key{Key: key})
	if res != nil {
		value = res.Data
		v = res.Version
	}

	return
}

func (c *Client) processGET(arg string) {
	ctx, done := c.ctx()
	defer done()

	value, _, err := c.Get(ctx, arg)
	if err != nil {
		fmt.Println("Error:", grpc.ErrorDesc(err))
		return
	}

	fmt.Printf("%s\n", value)
}

func (c *Client) processVERSION(arg string) {
	ctx, done := c.ctx()
	defer done()
	_, v, err := c.Get(ctx, arg)
	if err != nil || v.Matches(version.NoVersion) == nil {
		fmt.Println("0x0")
		return
	}

	fmt.Printf("0x%x\n", v.Hash)
}

func (c *Client) processMEMBERS(arg string) {
	ctx, done := c.ctx()
	defer done()
	values, err := c.client.Members(ctx, &api.Key{Key: arg})
	if err != nil {
		fmt.Println("Error:", grpc.ErrorDesc(err))
		return
	}

	fmt.Println(len(values.Data), "element(s)")
	for _, data := range values.Data {
		fmt.Printf("- %s\n", data)
	}
}

func (c *Client) processCONTAINS(arg string) {
	ctx, done := c.ctx()
	defer done()
	arg1, arg2, err := split2args(arg)
	if err != nil {
		fmt.Println("CONTAINS function expects two arguments: (container, element)")
		return
	}

	boolean, err := c.client.Contains(ctx, &api.KeyValue{Key: arg1, Value: []byte(arg2)})
	if err != nil {
		fmt.Println("Error:", grpc.ErrorDesc(err))
	}

	fmt.Println(boolean.Boolean)
}
