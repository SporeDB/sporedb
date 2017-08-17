package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/awnumar/memguard"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/db/drivers"
	"gitlab.com/SporeDB/sporedb/db/drivers/boltdb"
	endpoint "gitlab.com/SporeDB/sporedb/db/server"
	"gitlab.com/SporeDB/sporedb/myc"
	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

var fullSync *string
var recoverKeys *string
var storeDrivers map[string]drivers.Constructor

func init() {
	addDriver("boltdb", func(p string) (db.Store, error) {
		return boltdb.New(p)
	})
}

func addDriver(name string, c drivers.Constructor) {
	if storeDrivers == nil {
		storeDrivers = make(map[string]drivers.Constructor)
	}

	storeDrivers[name] = c
}

func getDriver(name string, path string) (db.Store, error) {
	if storeDrivers == nil || storeDrivers[name] == nil {
		fmt.Fprintln(os.Stderr, "Available database drivers:")
		for k := range storeDrivers {
			fmt.Fprintln(os.Stderr, "  *", k)
		}
		return nil, errors.New("unknown database driver: " + name)
	}

	return storeDrivers[name](path)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run a SporeDB node",
	Run: func(cmd *cobra.Command, args []string) {
		keyRing := getKeyRing()
		check(keyRing.UnlockPrivate(getPassword()))

		store, err := getDriver(viper.GetString("db.driver"), viper.GetString("db.path"))
		check(err)

		database := db.NewDB(store, viper.GetString("identity"), keyRing)
		loadPolicies(database)

		srv := &endpoint.Server{
			DB:     database,
			Listen: viper.GetString("api.listen"),
		}

		rawPeers := viper.GetStringSlice("mycelium.peers")
		peers := make([]protocol.Node, len(rawPeers))
		for i, p := range rawPeers {
			peers[i] = protocol.Node{Address: p}
		}

		mycelium, _ := myc.NewMycelium(&myc.MyceliumConfig{
			Listen: viper.GetString("mycelium.listen"),
			Peers:  peers,
			DB:     database,
		})

		go startRecover(database, mycelium)

		// Catch SIGINT and SIGTERM
		go func() {
			c := make(chan os.Signal, 2)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			for range c {
				fmt.Println("\nStopping SporeDB...")
				_ = store.Close()
				_ = mycelium.Close()
				database.Stop()
				memguard.SafeExit(0)
			}
		}()

		zap.L().Info("Listening",
			zap.String("type", "API"),
			zap.String("address", viper.GetString("api.listen")),
		)

		database.Start(false)
		err = srv.Serve()

		if err != nil {
			zap.L().Error("Unable to listen",
				zap.String("type", "API"),
				zap.Error(err),
			)
		}
	},
}

func startRecover(database *db.DB, mycelium *myc.Mycelium) {
	time.Sleep(5 * time.Second) // Artificial delay to ease start-up

	if *fullSync != "" {
		mycelium.StartFullSync(*fullSync)
	}

	for _, key := range strings.Split(*recoverKeys, ",") {
		if key == "" {
			continue
		}

		zap.L().Info("Recovery request",
			zap.String("key", key),
		)
		database.Messages <- &db.RecoverRequest{Key: key}
	}
}

func init() {
	RootCmd.AddCommand(serverCmd)

	recoverKeys = serverCmd.Flags().StringP("recover", "r", "", "set of keys to recover at startup (coma-separated)")
	fullSync = serverCmd.Flags().StringP("full-sync", "s", "", "identity of peer to ask for a full state-transfer")
}
