package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/db/drivers/rocksdb"
	endpoint "gitlab.com/SporeDB/sporedb/db/server"
	"gitlab.com/SporeDB/sporedb/myc"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run a SporeDB node",
	Run: func(cmd *cobra.Command, args []string) {
		store, err := rocksdb.New(viper.GetString("db.path"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		database := db.NewDB(store, viper.GetString("identity"))
		if viper.GetBool("db.none_policy") {
			fmt.Println("WARNING: The database has the \"none\" policy enabled,")
			fmt.Println("         this should be reserved for testing and development only")
			_ = database.AddPolicy(db.NonePolicy)
		}

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
}
