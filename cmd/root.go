package cmd

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var root = &cobra.Command{
	Use:   "moria",
	Short: "speak, friend, and enter",
}

func Execute() {
	err := root.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	var config string
	root.PersistentFlags().
		StringVar(&config, "config", "", "config file (default is config.json)")

	// OnInitialize runs after flag parsing, so --config is visible here.
	cobra.OnInitialize(func() {
		if config != "" {
			viper.SetConfigFile(config)
		} else {
			viper.AddConfigPath(".")
			viper.SetConfigType("json")
			viper.SetConfigName("config")
		}

		viper.SetEnvPrefix("MORIA")
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		viper.AutomaticEnv()
		if err := viper.ReadInConfig(); err == nil {
			slog.Info("using config file", "file", viper.ConfigFileUsed())
		}
	})

	root.AddCommand(serve)
}
