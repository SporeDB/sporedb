package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile *string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "sporedb",
	Short: "🍄 SporeDB project, a distributed, byzantine fault tolerant database.",
	Long:  ``,
}

func init() {
	cobra.OnInitialize(initConfig)
	cfgFile = RootCmd.PersistentFlags().StringP("config", "c", "", "config file (default is ./sporedb.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if *cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(*cfgFile)
	}

	viper.SetConfigName("sporedb") // name of config file (without extension)
	viper.AddConfigPath(".")       // adding home directory as first search path
	viper.AutomaticEnv()           // read in environment variables that match

	// If a config file is found, read it in.
	_ = viper.ReadInConfig()
}
