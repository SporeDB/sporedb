// Package cmd provides SporeDB CLI interface.
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	} else {
		viper.SetConfigName("sporedb") // name of config file (without extension)
		viper.AddConfigPath(".")       // adding home directory as first search path
	}

	// If a config file is found, read it in.
	var err = viper.ReadInConfig()
	check(err)
	viper.AutomaticEnv() // read in environment variables that match

	// Put default values
	if !viper.IsSet("db.driver") {
		viper.Set("db.driver", "boltdb")
	}

	// Init logging
	logEncoder := zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	logConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    logEncoder,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	l, _ := logConfig.Build()
	zap.ReplaceGlobals(l)
}
