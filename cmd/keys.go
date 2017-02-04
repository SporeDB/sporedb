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

func getKeyRing() sec.KeyRing {
	rawKeyRing, err := ioutil.ReadFile(viper.GetString("keyring"))
	check(err)

	keyRing := sec.NewKeyRingEd25519()
	check(keyRing.UnmarshalBinary(rawKeyRing))
	return keyRing
}

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage signature keys",
}

var keysInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create local keyring",
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

var keysExportCmd = &cobra.Command{
	Use:   "export [identity]",
	Short: "Export a public key from the keyring",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			args = []string{""}
		}

		keyRing := getKeyRing()
		data, err := keyRing.Export(args[0])
		check(err)
		fmt.Printf("%s", data)
	},
}

var keysImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a public key to the keyring",
	Run: func(cmd *cobra.Command, args []string) {
		check(errors.New("not yet implemented"))
	},
}

var keysRemoveCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove a public key from the keyring",
	Run: func(cmd *cobra.Command, args []string) {
		check(errors.New("not yet implemented"))
	},
}

var keysListCmd = &cobra.Command{
	Use:   "ls",
	Short: "List public keys from the keyring",
	Run: func(cmd *cobra.Command, args []string) {
		check(errors.New("not yet implemented"))
	},
}

func init() {
	keysCmd.AddCommand(keysInitCmd, keysExportCmd, keysImportCmd, keysRemoveCmd, keysListCmd)
	RootCmd.AddCommand(keysCmd)
}
