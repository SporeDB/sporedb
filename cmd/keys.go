package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

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

func saveKeyRing(keyRing sec.KeyRing) {
	data, err := keyRing.MarshalBinary()
	check(err)
	check(ioutil.WriteFile(viper.GetString("keyring"), data, 0600))
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
		saveKeyRing(keyRing)

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

var importIdentity *string
var importTrust *string

var keysImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a public key to the keyring",
	Run: func(cmd *cobra.Command, args []string) {
		identity := *importIdentity
		if identity == "" {
			check(errors.New("please provide identity flag"))
		}

		var lvl sec.TrustLevel
		switch *importTrust {
		case "none":
			lvl = sec.TrustNONE
		case "low":
			lvl = sec.TrustLOW
		case "high":
			lvl = sec.TrustHIGH
		case "ultimate":
			lvl = sec.TrustULTIMATE
		default:
			check(errors.New("unrecognized trust level"))
		}

		keyRing := getKeyRing()

		data, err := ioutil.ReadAll(os.Stdin)
		check(err)
		check(keyRing.Import(data, identity, lvl))

		saveKeyRing(keyRing)

		pub, _, _ := keyRing.GetPublic(identity)
		fmt.Printf("Imported new key for identity %s (%x)\n", identity, pub)
	},
}

var keysRemoveCmd = &cobra.Command{
	Use:   "rm [identity]",
	Short: "Remove a public key from the keyring",
	Run: func(cmd *cobra.Command, args []string) {
		check(errors.New("not yet implemented"))
	},
}

var keysListCmd = &cobra.Command{
	Use:   "ls [identity]",
	Short: "List public keys from the keyring",
	Run: func(cmd *cobra.Command, args []string) {
		check(errors.New("not yet implemented"))
	},
}

func init() {
	keysCmd.AddCommand(keysInitCmd, keysExportCmd, keysImportCmd, keysRemoveCmd, keysListCmd)
	RootCmd.AddCommand(keysCmd)

	importIdentity = keysImportCmd.Flags().StringP("identity", "i", "", "public key identity")
	importTrust = keysImportCmd.Flags().StringP("trust", "t", "low", "public key local trust (none, low, high, ultimate)")
}
