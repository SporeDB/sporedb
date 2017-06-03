package client

import (
	"context"
	"fmt"
	"sort"

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

// Members returns the slice of every element of a container.
func (c *Client) Members(ctx context.Context, key string) (values [][]byte, v *version.V, err error) {
	members, err := c.client.Members(ctx, &api.Key{Key: key})
	if members != nil {
		values = members.Data
		v = members.Version
	}

	return
}

// Contains returns wether or not a specific value is present in a container.
func (c *Client) Contains(ctx context.Context, key string, value []byte) (contains bool, err error) {
	boolean, err := c.client.Contains(ctx, &api.KeyValue{Key: key, Value: value})
	contains = boolean.Boolean
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
	values, _, err := c.Members(ctx, arg)
	if err != nil {
		fmt.Println("Error:", grpc.ErrorDesc(err))
		return
	}

	fmt.Println(len(values), "element(s)")

	strValues := make([]string, len(values))
	for i, data := range values {
		strValues[i] = string(data)
	}

	sort.Strings(strValues)

	for _, data := range strValues {
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

	contains, err := c.Contains(ctx, arg1, []byte(arg2))
	if err != nil {
		fmt.Println("Error:", grpc.ErrorDesc(err))
	}

	fmt.Println(contains)
}
