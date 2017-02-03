package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.com/SporeDB/sporedb/myc/sec"
)

func getPassword() string {
	password := viper.GetString("password")
	if len(password) == 0 {
		check(errors.New("Please provide a password through `PASSWORD` environment variable."))
	}
	return password
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configures a SporeDB node",
	Run: func(cmd *cobra.Command, args []string) {
		// Generate new KeyRing
		keyRing := sec.NewKeyRingEd25519()
		check(keyRing.CreatePrivate(getPassword()))

		// Save to disk
		data, err := keyRing.MarshalBinary()
		check(err)
		check(ioutil.WriteFile(viper.GetString("keyring"), data, 0600))

		// Print confirmation
		pub, _, _ := keyRing.GetPublic("")
		fmt.Printf("Generated new keyring (%x)\n", pub)
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
}
