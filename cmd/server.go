package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/db/drivers/rocksdb"
	endpoint "gitlab.com/SporeDB/sporedb/db/server"
	"gitlab.com/SporeDB/sporedb/myc"
)

var recoverKeys *string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run a SporeDB node",
	Run: func(cmd *cobra.Command, args []string) {
		keyRing := getKeyRing()
		check(keyRing.UnlockPrivate(getPassword()))

		store, err := rocksdb.New(viper.GetString("db.path"))
		check(err)

		database := db.NewDB(store, viper.GetString("identity"), keyRing)
		loadPolicies(database)

		srv := &endpoint.Server{
			DB:     database,
			Listen: viper.GetString("api.listen"),
		}

		rawPeers := viper.GetStringSlice("mycelium.peers")
		peers := make([]*myc.Node, len(rawPeers))
		for i, p := range rawPeers {
			peers[i] = &myc.Node{Address: p}
		}

		mycelium, _ := myc.NewMycelium(&myc.MyceliumConfig{
			Listen: viper.GetString("mycelium.listen"),
			Peers:  peers,
			DB:     database,
		})

		// Recover keys (optional)
		for _, key := range strings.Split(*recoverKeys, ",") {
			if key == "" {
				continue
			}

			fmt.Println("Sending recovery request for key `" + key + "`")
			database.Messages <- &db.RecoverRequest{Key: key}
		}

		// Catch SIGINT
		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			for range c {
				fmt.Println("\nStopping SporeDB...")
				_ = store.Close()
				_ = mycelium.Close()
				database.Stop()
				os.Exit(0)
			}
		}()

		fmt.Println("SporeDB is running on", viper.GetString("api.listen"))
		database.Start(false)
		_ = srv.Serve()
	},
}

func init() {
	RootCmd.AddCommand(serverCmd)

	recoverKeys = serverCmd.Flags().StringP("recover", "r", "", "set of keys to recover at startup (coma-separated)")
}
