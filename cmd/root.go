package cmd

import (
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "moria",
	Short: "speak, friend, and enter",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().
		String("config", "", "config file (default is .moria.json)")
}

func initializeConfig(cmd *cobra.Command) error {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	viper.SetEnvPrefix("moria")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	cfgFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("json")
		viper.SetConfigName(".moria")
	}

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return err
		}
	} else {
		slog.Info("using config file", "file", viper.ConfigFileUsed())
	}

	return viper.BindPFlags(cmd.Flags())
}
