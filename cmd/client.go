package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	endpoint "gitlab.com/SporeDB/sporedb/db/client"
)

var addrSrv *string
var timeoutSrv *time.Duration

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Run a SporeDB client in CLI",
	Run: func(cmd *cobra.Command, args []string) {
		cli := &endpoint.Client{
			Addr:    *addrSrv,
			Timeout: *timeoutSrv,
		}

		if err := cli.Connect(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		cli.CLI()
		cli.Close()
	},
}

func init() {
	RootCmd.AddCommand(clientCmd)
	addrSrv = clientCmd.Flags().StringP("server", "s", "localhost:4200", "server address")
	timeoutSrv = clientCmd.Flags().DurationP("timeout", "t", 10*time.Second, "connection timeout")
}
