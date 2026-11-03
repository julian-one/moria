package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Execute() error {
	root := RootCmd()
	if err := root.Execute(); err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	return nil
}

func RootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "moria",
		Short:        "Speak friend and enter",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return initConfig(cmd)
		},
	}
	root.PersistentFlags().String("config", "", "config file (default is .moria.json)")
	root.AddCommand(serveCmd())
	return root
}

func initConfig(cmd *cobra.Command) error {
	cfgFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return fmt.Errorf("read config flag: %w", err)
	}
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName(".moria")
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	var notFound viper.ConfigFileNotFoundError
	if err := viper.ReadInConfig(); err != nil && !errors.As(err, &notFound) {
		return fmt.Errorf("read config: %w", err)
	}
	return nil
}
