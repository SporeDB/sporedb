package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitlab.com/SporeDB/sporedb/db/api"

	"github.com/chzyer/readline"
	"google.golang.org/grpc"
)

// Client is the GRPC SporeDB client.
type Client struct {
	Addr    string
	Timeout time.Duration
	conn    *grpc.ClientConn
	client  api.SporeDBClient
	policy  string
}

// Connect proceeds to the GRPC connection step to the server.
func (c *Client) Connect() (err error) {
	c.conn, err = grpc.Dial(c.Addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(c.Timeout))
	if err != nil {
		return err
	}

	c.client = api.NewSporeDBClient(c.conn)
	return nil
}

// Close closes the GRPC connection to the server.
func (c *Client) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// CLI starts a command line interface to dial with the GRPC server (debug and maintenance).
func (c *Client) CLI() {
	fmt.Println("SporeDB client is connected and ready to execute your luscious instructions!")
	rl, err := readline.New(c.Addr + "> ")
	if err != nil {
		return
	}
	defer func() { _ = rl.Close() }()

	m := c.getCLIMap()

	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}

		args := strings.SplitN(line, " ", 2)
		cmd := strings.ToUpper(args[0])

		f, ok := m[cmd]
		if !ok {
			fmt.Println("Invalid command")
			continue
		}

		arg := ""
		if len(args) > 1 {
			arg = args[1]
		}
		f(arg)
	}
}

func (c *Client) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.Timeout)
}
